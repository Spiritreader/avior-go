package worker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	jobLog := new(joblog.Data)
	_ = glg.Infof("processing job %s", job.Path)
	mediaFile := &media.File{Path: job.Path, Name: job.Name, Subtitle: job.Subtitle, EncodeParams: job.CustomParameters}
	err := mediaFile.Update()
	if err != nil {
		Resume(resumeChan)
		return
	}
	fmt.Println(mediaFile.Path)
	fmt.Println(mediaFile.OutName())
	jobLog.AddFileProperties(*mediaFile)
	res := runModules(*jobLog, *mediaFile)
	switch res {
	case comparator.KEEP:
		_ = jobLog.AppendTo(mediaFile.Path + ".INFO.log")
		_ = jobLog.AppendTo(filepath.Join("log", "skipped.log"))
		Resume(resumeChan)
		return
	}

	duplicates := checkForDuplicates(mediaFile)
	if dupeLen := len(duplicates); dupeLen > 0 {
		_ = glg.Infof("found %d duplicates, selecting first", dupeLen)
		res, moduleName := runDupeModules(*jobLog, *mediaFile, duplicates[0])
		obsoleteDir := filepath.Join(config.Instance().Local.ObsoletePath, consts.OBSOLETE_DIR)
		err := move(duplicates[0], obsoleteDir, &moduleName)
		if err != nil {
			_ = glg.Errorf("can't continue without moving duplicate files, skipping job")
			Resume(resumeChan)
			return
		}
		switch res {
		case comparator.KEEP:
			existDir := filepath.Join(filepath.Dir(mediaFile.Path), consts.EXIST_DIR)
			err = move(*mediaFile, existDir, nil)
			if err != nil {
				_ = glg.Warnf("couldn't move source files to exist directory, err: %s", err)
			}
			_ = jobLog.AppendTo(mediaFile.Path + ".INFO.log")
			_ = jobLog.AppendTo(filepath.Join("log", "skipped.log"))
			Resume(resumeChan)
			return
		}
	}

	_ = glg.Infof("encoding file %s", mediaFile.Path)
	stats, err := encoder.Encode(*mediaFile, 0, 0, false)
	if err != nil || stats.ExitCode != 0 {
		_ = glg.Warnf("encode failed, retrying")
		stats, err = encoder.Encode(*mediaFile, 0, 0, true)
		if err != nil {
			Resume(resumeChan)
			return
		}
	}
	_ = glg.Infof("encode to %s done in %d", stats.OutputPath, stats.Duration)
	jobLog.Add("")
	jobLog.Add(fmt.Sprintf("Output Path: %s", stats.OutputPath))
	jobLog.Add(fmt.Sprintf("Duration: %d", stats.Duration))
	jobLog.Add(fmt.Sprintf("Parameters: %s", stats.Call))
	_ = jobLog.AppendTo(filepath.Join("log", "processed.log"))
	_ = jobLog.AppendTo(mediaFile.LogPaths[0])

	doneDir := filepath.Join(filepath.Dir(mediaFile.Path), consts.DONE_DIR)
	err = move(*mediaFile, doneDir, nil)
	if err != nil {
		_ = glg.Warnf("couldn't move source files to done directory, err: %s", err)
	}
	Resume(resumeChan)
}

func Resume(resumeChan chan string) {
	select {
	case resumeChan <- consts.RESUME:
		_ = glg.Log("sending resume event")
	default:
		_ = glg.Log("resume event already waiting for consumption")
	}
}

func runModules(jobLog joblog.Data, fileNew media.File) string {
	jobLog.Add("Module Results:")
	modules := comparator.InitStandaloneModules()
	for idx := range modules {
		name, result, message := modules[idx].Run(fileNew)
		_ = glg.Infof("%s: %s - %s", name, result, message)
		jobLog.Add(fmt.Sprintf("%s: %s - %s", name, result, message))

		switch result {
		case comparator.NOCH:
			continue
		case comparator.KEEP:
			return comparator.KEEP
		case comparator.REPL:
			return comparator.REPL
		}
	}
	return comparator.NOCH
}

func runDupeModules(jobLog joblog.Data, fileNew media.File, fileDup media.File) (string, string) {
	jobLog.Add("Dupe Module Results:")
	modules := comparator.InitDupeModules()
	for idx := range modules {
		name, result, message := modules[idx].Run(fileNew, fileDup)
		_ = glg.Infof("%s: %s - %s", name, result, message)
		jobLog.Add(fmt.Sprintf("%s: %s - %s", name, result, message))

		switch result {
		case comparator.NOCH:
			continue
		case comparator.KEEP:
			return comparator.KEEP, name
		case comparator.REPL:
			return comparator.REPL, name
		}
	}
	return comparator.NOCH, "none"
}

func move(file media.File, dstDir string, moduleName *string) error {
	_, err := os.Stat(dstDir)
	if os.IsNotExist(err) {
		_ = os.Mkdir(dstDir, 0777)
	}
	toMovePaths := make(map[string]string, 0)
	fileOut := strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
	if moduleName != nil {
		fileOut += *moduleName
	}
	fileOut += filepath.Ext(file.Path)
	fileOut = filepath.Join(dstDir, fileOut)
	toMovePaths[file.Path] = fileOut
	for _, log := range file.LogPaths {
		logOut := strings.TrimSuffix(filepath.Base(log), filepath.Ext(log))
		if moduleName != nil {
			logOut += *moduleName
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

// checkForDuplicates retrieves all duplicates for the given file,
//
// given a slice of media paths that should be searched
func checkForDuplicates(file *media.File) []media.File {
	cfg := config.Instance()
	state.FileWalker.Position = 0
	matches := make([]media.File, 0)
	for idx, path := range cfg.Local.MediaPaths {
		state.FileWalker.Directory = path
		_ = glg.Infof("scanning directory (%d/%d): %s", idx, len(cfg.Local.MediaPaths), path)
		dir_matches, count, _ := traverseDir(file, path)
		state.FileWalker.Position += count
		matches = append(matches, dir_matches...)
	}
	cfg.Local.EstimatedLibSize = state.FileWalker.Position
	_ = config.Save()
	return matches
}

func traverseDir(file *media.File, path string) ([]media.File, int, error) {
	counter := 0
	matches := make([]media.File, 0)
	err := godirwalk.Walk(path, &godirwalk.Options{
		Unsorted: true,
		Callback: func(path string, de *godirwalk.Dirent) error {
			if de.IsDir() && strings.HasPrefix(de.Name(), ".") {
				_ = glg.Logf("skipping hidden dir %s", path)
				return errors.New("directory ignored")
			}
			if !de.IsDir() && strings.Contains(de.Name(), file.OutName()) {
				file := &media.File{Path: path}
				_ = glg.Infof("found duplicate: %s", path)
				matches = append(matches, *file)
			}
			if !de.IsDir() && strings.HasSuffix(de.Name(), config.Instance().Local.Ext) {
				counter++
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
	return matches, counter, nil
}
