package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Spiritreader/avior-go/comparator"
	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/encoder"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/Spiritreader/avior-go/joblog"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/Spiritreader/avior-go/tools"
	"github.com/karrick/godirwalk"
	"github.com/kpango/glg"
)

var state *globalstate.Data = globalstate.Instance()

func ProcessJob(dataStore *db.DataStore, client *structs.Client, job *structs.Job, resumeChan chan string) {
	state.InFile = job.Path
	jobLog := new(joblog.Data)
	_ = glg.Infof("processing job %s", job.Path)

	//reset global state after job, allow resume without pause
	defer func() {
		lineOut := state.Encoder.LineOut
		state.Clear()
		state.Encoder.LineOut = lineOut
		Resume(resumeChan)
	}()

	//populate media file
	mediaFile := &media.File{Path: job.Path, Name: job.Name, Subtitle: job.Subtitle, CustomParams: job.CustomParameters}
	err := mediaFile.Update()
	if err != nil {
		_ = glg.Errorf("couldn't parse media file: %s", err)
		return
	}
	_ = glg.Logf("input file: %s", mediaFile.Path)
	_ = glg.Logf("trimmed name: %s", mediaFile.OutName())
	jobLog.AddFileProperties(*mediaFile)

	// run single file moduels
	jobLog.Add("")
	res := runModules(jobLog, *mediaFile)
	switch res {
	case comparator.DISC:
		appendJobTemplate(*job, jobLog, false)
		writeSkippedLog(mediaFile, jobLog)
		return
	}

	// check for duplicates and run modules
	var redirectDir *string = nil
	duplicates, err := checkForDuplicates(mediaFile)
	if err != nil {
		_ = glg.Errorf("duplicate scan failed, please fix. Pausing service to prevent unwanted behavior")
		state.Paused = true
		appendJobTemplate(*job, jobLog, false)
		writeSkippedLog(mediaFile, jobLog)
		return
	}
	if dupeLen := len(duplicates); dupeLen > 0 {
		_ = glg.Infof("found %d duplicates, selecting first", dupeLen)
		err = duplicates[0].Update()
		if err != nil {
			_ = glg.Warnf("couldn't parse duplicate log file: %s", err)
		}

		// run dupe file modules and prevent replacement if necessary
		jobLog.Add("")
		res, moduleName := runDupeModules(jobLog, *mediaFile, duplicates[0])
		switch res {
		case comparator.DISC, comparator.NOCH:
			appendJobTemplate(*job, jobLog, true)
			writeSkippedLog(mediaFile, jobLog)
			if filepath.Dir(mediaFile.Path) == consts.EXIST_DIR {
				return
			}
			existDir := filepath.Join(filepath.Dir(mediaFile.Path), consts.EXIST_DIR)
			err = moveMediaFile(*mediaFile, existDir, nil)
			if err != nil {
				_ = glg.Warnf("couldn't move source media file to exist directory, err: %s", err)
			}
			err = moveLogs(*mediaFile, existDir, nil)
			if err != nil {
				_ = glg.Warnf("couldn't move source log files to exist directory, err: %s", err)
			}
			return
		}

		// if dupe file is eligible for replacement, move it to the .obsolete dir
		obsoleteDir := filepath.Join(config.Instance().Local.ObsoletePath, consts.OBSOLETE_DIR)
		errM := moveMediaFile(duplicates[0], obsoleteDir, &moduleName)
		errL := moveLogs(duplicates[0], obsoleteDir, &moduleName)
		if errM != nil || errL != nil {
			msg := "can't continue without moving duplicate files, skipping job"
			_ = glg.Errorf(msg)
			jobLog.Add("error:")
			if errM != nil {
				jobLog.Add(errM.Error())
			}
			if errL != nil {
				jobLog.Add(errL.Error())
			}
			jobLog.Add(msg)
			appendJobTemplate(*job, jobLog, false)
			writeSkippedLog(mediaFile, jobLog)
			return
		}
		duplicateDir := filepath.Dir(duplicates[0].Path)
		redirectDir = &duplicateDir
	}

	jobLog.Add("")
	// encode with one retry that overwrites (in case the old one failed)
	_ = glg.Infof("encoding file %s", mediaFile.Path)
	_ = glg.Logf("media struct: %+v", *mediaFile)
	jobLog.Add("Encoder Info:")
	stats, err := encoder.Encode(*mediaFile, 0, 0, false, redirectDir)
	jobLog.Add(fmt.Sprintf("OutputPath: %s", state.Encoder.OutPath))

	if err != nil {
		if err.Error() == "no resolution tag found" {
			appendJobTemplate(*job, jobLog, false)
			writeSkippedLog(mediaFile, jobLog)
			return
		}
		if stats.ExitCode == 1 {
			_ = glg.Errorf("encoding of %s failed, err: %s (file already exists)", state.Encoder.OutPath, err)
			_ = glg.Infof("skipping file")
			jobLog.Add("encode error: ffmpeg exit code 1 (file already exists)")
			appendJobTemplate(*job, jobLog, false)
			writeSkippedLog(mediaFile, jobLog)
			_ = jobLog.AppendTo(mediaFile.Path+".INFO.log", false, false)
			return
		}
		_ = glg.Warnf("encode failed, retrying")
		// allow overwrite for retry to avoid it failing imdmediately
		stats, err = encoder.Encode(*mediaFile, 0, 0, true, redirectDir)
		if err != nil {
			_ = glg.Errorf("re-encoding of %s failed, err: %s", job.Path, err)
			_ = glg.Infof("skipping file")
			jobLog.Add("\nencode retry error: " + err.Error())
			appendJobTemplate(*job, jobLog, false)
			writeSkippedLog(mediaFile, jobLog)
			_ = jobLog.AppendTo(mediaFile.Path+".INFO.log", false, false)
			return
		}
	}
	_ = glg.Infof("encode to %s done in %s", stats.OutputPath, stats.Duration)
	jobLog.Add(fmt.Sprintf("Duration: %s", stats.Duration))
	jobLog.Add(fmt.Sprintf("Parameters: %s", stats.Call))
	_ = jobLog.AppendTo(filepath.Join(globalstate.ReflectionPath(), "log", "processed.log"), false, true)
	_ = jobLog.AppendTo(mediaFile.LogPaths[0], true, false)
	

	// move files, cleanup
	doneDir := filepath.Join(filepath.Dir(mediaFile.Path), consts.DONE_DIR)
	err = moveMediaFile(*mediaFile, doneDir, nil)
	if err != nil {
		_ = glg.Errorf("couldn't move source media file to done directory, err: %s", err)
	}
	err = copyLogsToEncOut(*mediaFile, filepath.Dir(stats.OutputPath))
	if err != nil {
		_ = glg.Errorf("couldn't copy source log files to encoded file directory, err: %s", err)
	}
	err = moveLogs(*mediaFile, doneDir, nil)
	if err != nil {
		_ = glg.Errorf("couldn't move source media file to done directory, err: %s", err)
	}
}

func Resume(resumeChan chan string) {
	select {
	case resumeChan <- consts.RESUME:
		_ = glg.Log("sending resume event")
	default:
		_ = glg.Log("resume event already waiting for consumption")
	}
}

func appendJobTemplate(job structs.Job, jobLog *joblog.Data, moved bool) {
	job.CustomParameters = append(job.CustomParameters, consts.MODULE_FLAG_SKIP)
	if moved {
		job.Path = filepath.Join(filepath.Dir(job.Path), consts.EXIST_DIR, filepath.Base(job.Path))
	}
	bytes, err := json.MarshalIndent([]structs.Job{job}, "", "  ")
	if err != nil {
		_ = glg.Warnf("couldn't attach database job to job log, err %s", err)
	} else {
		jobLog.Add("Job Database Template: ")
		jobLog.Add(string(bytes))
	}
}

func runModules(jobLog *joblog.Data, fileNew media.File) string {
	jobLog.Add("Module Results:")
	if fileNew.AllowReplacement {
		jobLog.Add("AllowReplacement: manual user override")
		_ = glg.Info("modules: manual user override, allow replacement")
		return comparator.REPL
	}
	modules := comparator.InitStandaloneModules()
	for idx := range modules {
		name, result, message := modules[idx].Run(fileNew)
		_ = glg.Infof("%s: %s - %s", name, result, message)
		jobLog.Add(fmt.Sprintf("%s: %s - %s", name, result, message))

		switch result {
		case comparator.NOCH:
			continue
		case comparator.DISC:
			return comparator.DISC
		case comparator.REPL:
			return comparator.REPL
		}
	}
	return comparator.NOCH
}

func runDupeModules(jobLog *joblog.Data, fileNew media.File, fileDup media.File) (string, string) {
	jobLog.Add("Dupe Module Results:")
	jobLog.Add(fmt.Sprintf("DupPath: %s", fileDup.Path))
	if fileNew.AllowReplacement {
		jobLog.Add("AllowReplacement: manual user override")
		return comparator.REPL, "AllowReplacement"
	}
	modules := comparator.InitDupeModules()
	for idx := range modules {
		name, result, message := modules[idx].Run(fileNew, fileDup)
		_ = glg.Infof("%s: %s - %s", name, result, message)
		jobLog.Add(fmt.Sprintf("%s: %s - %s", name, result, message))

		switch result {
		case comparator.NOCH:
			continue
		case comparator.DISC:
			return comparator.DISC, name
		case comparator.REPL:
			state.Encoder.ReplacementReason = fmt.Sprintf("%s: %s - %s", name, result, message)
			return comparator.REPL, name
		}
	}
	return comparator.NOCH, "none"
}

func writeSkippedLog(mediaFile *media.File, jobLog *joblog.Data) {
	mediaFile.LogPaths = append(mediaFile.LogPaths, mediaFile.Path+".INFO.log")
	_ = jobLog.AppendTo(mediaFile.Path+".INFO.log", false, false)
	_ = jobLog.AppendTo(filepath.Join(globalstate.ReflectionPath(), "log", "skipped.log"), false, true)
}

func moveMediaFile(file media.File, dstDir string, moduleName *string) error {
	_, err := os.Stat(dstDir)
	if os.IsNotExist(err) {
		_ = os.Mkdir(dstDir, 0777)
	}
	fileOut := strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
	if moduleName != nil {
		fileOut += " "
		fileOut += *moduleName
		fileOut += " " + time.Now().Format("2006-01-02 1504")
	}
	fileOut += filepath.Ext(file.Path)
	fileOut = filepath.Join(dstDir, fileOut)
	err = tools.MoppyFile(file.Path, fileOut, true)
	if err != nil {
		return err
	}
	return nil
}

func moveLogs(file media.File, dstDir string, moduleName *string) error {
	_, err := os.Stat(dstDir)
	if os.IsNotExist(err) {
		_ = os.Mkdir(dstDir, 0777)
	}
	toMovePaths := make(map[string]string)
	for _, log := range file.LogPaths {
		logOut := strings.TrimSuffix(filepath.Base(log), filepath.Ext(log))
		if moduleName != nil {
			logOut += " "
			logOut += *moduleName
			logOut += " " + time.Now().Format("2006-01-02 1504")
		}
		logOut += filepath.Ext(log)
		logOut = filepath.Join(dstDir, logOut)
		toMovePaths[log] = logOut
	}
	for src, dst := range toMovePaths {
		err := tools.MoppyFile(src, dst, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func copyLogsToEncOut(file media.File, dstDir string) error {
	_, err := os.Stat(dstDir)
	if os.IsNotExist(err) {
		_ = os.Mkdir(dstDir, 0777)
	}
	toMovePaths := make(map[string]string)
	for _, log := range file.LogPaths {
		logOut := file.OutName()
		logOut += filepath.Ext(log)
		logOut = filepath.Join(dstDir, logOut)
		toMovePaths[log] = logOut
	}
	for src, dst := range toMovePaths {
		err := tools.MoppyFile(src, dst, false)
		if err != nil {
			return err
		}
	}
	return nil
}

// checkForDuplicates retrieves all duplicates for the given file,
//
// given a slice of media paths that should be searched
func checkForDuplicates(file *media.File) ([]media.File, error) {
	cfg := config.Instance()
	state.FileWalker.Active = true
	defer func() {
		state.FileWalker.Active = false
	}()
	counter := 0
	state.FileWalker.Position = 0
	state.FileWalker.LibSize = cfg.Local.EstimatedLibSize
	matches := make([]media.File, 0)
	for idx, path := range cfg.Local.MediaPaths {
		state.FileWalker.Directory = path
		_ = glg.Infof("scanning directory (%d/%d): %s", idx+1, len(cfg.Local.MediaPaths), path)
		dir_matches, count, err := traverseDir(file, path)
		if err != nil {
			return []media.File{}, err
		}
		counter += count
		matches = append(matches, dir_matches...)
	}
	cfg.Local.EstimatedLibSize = state.FileWalker.Position
	state.FileWalker.Position = 0
	_ = config.Save()
	return matches, nil
}

func traverseDir(file *media.File, path string) ([]media.File, int, error) {
	matches := make([]media.File, 0)
	err := godirwalk.Walk(path, &godirwalk.Options{
		Unsorted: true,
		Callback: func(path string, de *godirwalk.Dirent) error {
			if de.IsDir() && strings.HasPrefix(de.Name(), ".") {
				return errors.New("directory ignored")
			}
			if !de.IsDir() && de.Name() == (file.OutName()+config.Instance().Local.Ext) {
				file := &media.File{Path: path}
				_ = glg.Infof("found duplicate: %s", path)
				matches = append(matches, *file)
			}
			if !de.IsDir() && strings.HasSuffix(de.Name(), config.Instance().Local.Ext) {
				if state.FileWalker.Position%1000 == 0 {
					_ = glg.Logf("current dir: %s, position: %d/%d",
						filepath.Dir(path), state.FileWalker.Position, state.FileWalker.LibSize)
				}
				state.FileWalker.Position++
			}
			return nil
		},
		ErrorCallback: func(path string, err error) godirwalk.ErrorAction {
			if err != nil && err.Error() != "directory ignored" {
				_ = glg.Warnf("could not read %s, skipping: %s", path, err)
			}
			return godirwalk.SkipNode
		},
	})
	if err != nil {
		_ = glg.Errorf("error traversing directory %s: %s", path, err)
		return nil, 0, err
	}
	return matches, state.FileWalker.Position, nil
}
