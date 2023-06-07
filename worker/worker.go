package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Spiritreader/avior-go/cache"
	"github.com/Spiritreader/avior-go/comparator"
	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/encoder"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/Spiritreader/avior-go/joblog"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/redis"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/Spiritreader/avior-go/tools"
	"github.com/karrick/godirwalk"
	"github.com/kpango/glg"
)

var (
	state                  *globalstate.Data = globalstate.Instance()
	previousEncoderLineOut []string
)

func ProcessJob(dataStore *db.DataStore, client *structs.Client, job *structs.Job, resumeChan chan string) {
	cfg := config.Instance()
	state.InFile = job.Path
	jobLog := new(joblog.Data)
	redis := redis.Get()
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

	// run single file modules
	jobLog.Add("")
	res := runModules(jobLog, *mediaFile)
	switch res {
	case comparator.DISC:
		appendJobTemplate(*job, jobLog, false)
		writeSkippedLog(mediaFile, jobLog, false)
		return
	}

	// check for duplicates and run modules
	var redirectDir *string = nil
	var obsoleteMovedLogPaths map[string]string = nil
	var obsoleteMovedFilePath map[string]string = nil
	duplicates, err := checkForDuplicates(mediaFile)
	if err != nil {
		_ = glg.Errorf("duplicate scan failed, please fix. Pausing service to prevent unwanted behavior: %s", err)
		state.Paused = true
		state.PauseReason = consts.PAUSE_REASON_DUPLICATE_SCAN
		appendJobTemplate(*job, jobLog, false)
		writeSkippedLog(mediaFile, jobLog, false)
		return
	}
	if dupeLen := len(duplicates); dupeLen > 0 {
		_ = glg.Infof("found %d duplicates, selecting first", dupeLen)

		// check if duplicate file actually exists
		if _, err := os.Stat(duplicates[0].Path); os.IsNotExist(err) {
			_ = glg.Warnf("duplicate file %s doesn't exist on disk, skipping", duplicates[0].Path)
			appendJobTemplate(*job, jobLog, false)
			writeSkippedLog(mediaFile, jobLog, false)
			return
		}

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
			writeSkippedLog(mediaFile, jobLog, false)
			if filepath.Dir(mediaFile.Path) == consts.EXIST_DIR {
				return
			}
			existDir := filepath.Join(filepath.Dir(mediaFile.Path), consts.EXIST_DIR)
			err, _ = moveMediaFile(*mediaFile, existDir, nil)
			if err != nil {
				_ = glg.Warnf("couldn't move source media file to exist directory, err: %s", err)
			}
			err, _ = moveLogs(*mediaFile, existDir, nil)
			if err != nil {
				_ = glg.Warnf("couldn't move source log files to exist directory, err: %s", err)
			}
			return
		}

		// if dupe file is eligible for replacement, move it to the .obsolete dir
		obsoleteDir := filepath.Join(cfg.Local.ObsoletePath, consts.OBSOLETE_DIR)
		var errL error = nil
		var errM error
		errM, obsoleteMovedFilePath = moveMediaFile(duplicates[0], obsoleteDir, &moduleName)
		if errM != nil {
			// roll back media file move if log move fails to ensure nothing was moved erroneously
			// map is structured in "original path": "moved path" pairs, so swap directions to move back
			src := obsoleteMovedFilePath[duplicates[0].Path]
			errRollback := tools.MoppyFile(src, duplicates[0].Path, true)
			if errRollback != nil {
				msg := fmt.Sprintf("failed to roll back media file move for %s, err: %s", src, errRollback)
				glg.Errorf(msg)
				jobLog.Add(fmt.Sprintf("error: %s", msg))
			}
		} else {
			errL, obsoleteMovedLogPaths = moveLogs(duplicates[0], obsoleteDir, &moduleName)
			if errL != nil {
				// roll back log move if media file move fails to ensure nothing was moved erroneously
				// map is structured in "original path": "moved path" pairs, so swap directions to move back
				for dst, src := range obsoleteMovedLogPaths {
					errRollback := tools.MoppyFile(src, dst, true)
					if errRollback != nil {
						msg := fmt.Sprintf("failed to roll back media file move for %s, err: %s", src, errRollback)
						glg.Errorf(msg)
						jobLog.Add(fmt.Sprintf("error: %s", msg))
					}
				}
			}
		}

		// cancel operation if any move failed
		if errM != nil || errL != nil {
			msg := "can't continue without moving duplicate files, skipping job"
			_ = glg.Errorf(msg)
			jobLog.Add("error:")
			if errM != nil {
				jobLog.Add(fmt.Sprintf("error: %s", errM.Error()))
			}
			if errL != nil {
				jobLog.Add(fmt.Sprintf("error: %s", errL.Error()))
			}
			jobLog.Add(msg)
			appendJobTemplate(*job, jobLog, false)
			writeSkippedLog(mediaFile, jobLog, false)
			return
		}
		// when everything is successful, set the redirect dir to the dupe dir so the media file encode
		// destination is the same as the dupe file
		duplicateDir := filepath.Dir(duplicates[0].Path)
		redirectDir = &duplicateDir
	}

	jobLog.Add("")
	// encode with one retry that overwrites (in case the old one failed)
	_ = glg.Infof("encoding file %s", mediaFile.Path)
	_ = glg.Logf("media struct: %+v", *mediaFile)
	jobLog.Add("Encoder Info:")

	// invalidate cache in non-redis mode as it won't be recent anymore after encoding a job
	if !redis.Handle.Running(){
		cache.Instance().Library.Valid = false
	}

	previousEncoderLineOut = make([]string, 0)
	stats, err := encoder.Encode(*mediaFile, 0, 0, false, redirectDir)
	jobLog.Add(fmt.Sprintf("OutputPath: %s", state.Encoder.OutPath))

	if err != nil {
		isBricked := false
		if errors.Is(err, encoder.ErrNoTag) {
			jobLog.Add(fmt.Sprintf("no encoder config found for tag %s, file %s", mediaFile.Resolution.Tag, mediaFile.Path))
			isBricked = true
		} else if stats.ExitCode == 108 {
			_ = glg.Errorf("encoding of %s failed, err: %s (file already exists)", state.Encoder.OutPath, err)
			jobLog.Add("encode error: ffmpeg exit code 1 (file already exists)")
			isBricked = true
		} else if stats.ExitCode == 107 {
			_ = glg.Errorf("encoding of %s failed, err: %s (file already exists)", state.Encoder.OutPath, err)
			jobLog.Add("encode error: os.IsNotExist returned false and overwrite has been disabled")
			isBricked = true
		}
		if isBricked {
			_ = glg.Infof("skipping file")
			appendJobTemplate(*job, jobLog, false)
			writeSkippedLog(mediaFile, jobLog, false)
			if redirectDir != nil {
				rollbackAllDupMoves(jobLog, obsoleteMovedFilePath, obsoleteMovedLogPaths)
			}
			if (cfg.Local.PauseOnEncodeError) {
				state.Paused = true
				state.PauseReason = consts.PAUSE_REASON_ENCODE_ERROR
			}
			return
		}

		// if the error is non bricking, attempt a re-encode
		_ = glg.Warnf("encode failed, retrying")
		// allow overwrite for retry to avoid it failing immediately
		var errRetry error
		previousEncoderLineOut = state.Encoder.LineOut
		stats, errRetry = encoder.Encode(*mediaFile, 0, 0, true, redirectDir)
		if errRetry != nil {
			_ = glg.Errorf("retrying encode failed. ffmpeg output has been appended to info log, file path: %s, err: %s", job.Path, errRetry)
			_ = glg.Infof("skipping file")
			jobLog.Add("Encode retry error: " + errRetry.Error())
			appendJobTemplate(*job, jobLog, false)
			writeSkippedLog(mediaFile, jobLog, true)
			if redirectDir != nil {
				rollbackAllDupMoves(jobLog, obsoleteMovedFilePath, obsoleteMovedLogPaths)
			}
			if (cfg.Local.PauseOnEncodeError) {
				state.Paused = true
				state.PauseReason = consts.PAUSE_REASON_ENCODE_ERROR
			}
			return
		}
	}

	_ = glg.Infof("encode to %s done in %s", stats.OutputPath, stats.Duration)
	jobLog.Add(fmt.Sprintf("Duration: %s", stats.Duration))
	jobLog.Add(fmt.Sprintf("Parameters: %s", stats.Call))
	_ = jobLog.AppendTo(filepath.Join(globalstate.ReflectionPath(), "log", "processed.log"), false, true)

	// sanitize log before appending encoding information to remove previous encoding data
	mediaFile.SanitizeLog()
	_ = jobLog.AppendTo(mediaFile.LogPaths[0], true, false)

	// move source files, cleanup
	doneDir := filepath.Join(filepath.Dir(mediaFile.Path), consts.DONE_DIR)
	err, _ = moveMediaFile(*mediaFile, doneDir, nil)
	if err != nil {
		_ = glg.Errorf("couldn't move source media file to done directory, err: %s", err)
	}
	err = copyLogsToEncOut(*mediaFile, filepath.Dir(stats.OutputPath))
	if err != nil {
		_ = glg.Errorf("couldn't copy source log files to encoded file directory, err: %s", err)
	}
	err, _ = moveLogs(*mediaFile, doneDir, nil)
	if err != nil {
		_ = glg.Errorf("couldn't move source media file to done directory, err: %s", err)
	}

	// broadcast job if redis is enabled
	if (redis.Handle.Running()) {
		_ = glg.Infof("redis: broadcasting job %s", stats.OutputPath)
		err := redis.Handle.PushMessage(stats.OutputPath)
		if err != nil {
			_ = glg.Warnf("redis: couldn't broadcast job, err: %s", err)
		}
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

// rolls back existing files to the original location
func rollbackAllDupMoves(jobLog *joblog.Data, fileRollbackPath map[string]string, logsRollbackPaths map[string]string) {
	// rollback file move
	// originally moves the file from the source to the destination, since we want to reverse the process we need to
	// switch the destination and source while iterating over the map
	for destination, source := range fileRollbackPath {
		err := tools.MoppyFile(source, destination, true)
		if err != nil {
			msg := fmt.Sprintf("couldn't rollback media file %s, err %s", source, err)
			glg.Errorf(msg)
			glg.Warnf("log files have not been rolled back due to previous failure")
			jobLog.Add(fmt.Sprintf("error: %s", msg))
			return
		}
	}
	// destination and source are switched because the logsRollbackPaths are outputted from moveLogs, which
	// originally moves them from the source to the destination, since we want to reverse the process we need to
	// switch the destination and source while iterating over the map
	for destination, source := range logsRollbackPaths {
		err := tools.MoppyFile(source, destination, true)
		if err != nil {
			msg := fmt.Sprintf("couldn't rollback log %s, err %s", source, err)
			glg.Errorf(msg)
			jobLog.Add(fmt.Sprintf("error: %s", msg))
		}
	}
}

func appendJobTemplate(job structs.Job, jobLog *joblog.Data, moved bool) {
	skipFlagPresent := false
	for _, line := range job.CustomParameters {
		if line == consts.MODULE_FLAG_SKIP {
			skipFlagPresent = true
			break
		}
	}
	if !skipFlagPresent {
		job.CustomParameters = append(job.CustomParameters, consts.MODULE_FLAG_SKIP)
	}
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

func appendFfmpegOutput(jobLog *joblog.Data, encoderState globalstate.Encoder) {
	jobLog.Add("FFmpeg Output:")
	hasData := false
	if len(previousEncoderLineOut) > 0 {
		jobLog.Add("Initial Attempt:")
		jobLog.Add(fmt.Sprintf("%v", previousEncoderLineOut))
		hasData = true
	}
	if len(encoderState.LineOut) > 0 {
		jobLog.Add("\nRetry attempt:")
		jobLog.Add(fmt.Sprintf("%v", encoderState.LineOut))
		hasData = true
	}
	if !hasData {
		jobLog.Add("No output")
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

// Writes the skipped logs to the skipped log file and the media file info log.
//
// If the withFfmpegOut flag is set, the ffmpeg output will be appended to the info log, but not to the skipped log.
func writeSkippedLog(mediaFile *media.File, jobLog *joblog.Data, withFfmpegOut bool) {
	mediaFile.LogPaths = append(mediaFile.LogPaths, mediaFile.Path+".INFO.log")
	if withFfmpegOut {
		jobLogWithFfmpeg := *jobLog
		appendFfmpegOutput(&jobLogWithFfmpeg, state.Encoder)
		_ = jobLogWithFfmpeg.AppendTo(mediaFile.Path+".INFO.log", false, false)
	} else {
		_ = jobLog.AppendTo(mediaFile.Path+".INFO.log", false, false)
	}
	_ = jobLog.AppendTo(filepath.Join(globalstate.ReflectionPath(), "log", "skipped.log"), false, true)
}

// Moves the mediafile to the dstDir location.
//
// If moduleName is specified, it will be appended to the filename.
//
// # Returns
//
// the new path of the file it was moved to
//
// In case of an error, the path will still be returned for rollback purposes, but the error will be non-nil
func moveMediaFile(file media.File, dstDir string, moduleName *string) (error, map[string]string) {
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
		return err, map[string]string{file.Path: fileOut}
	}
	return nil, map[string]string{file.Path: fileOut}
}

// Moves the logs of the mediafile to the dstDir location.
//
// If moduleName is specified, it will be appended to the filename.
//
// # Returns the new paths of the file it was moved to as a map of old path to new path
//
// In case of an error, the path of the failed move will still be returned for rollback purposes
func moveLogs(file media.File, dstDir string, moduleName *string) (error, map[string]string) {
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
			return err, map[string]string{src: dst}
		}
	}
	return nil, toMovePaths
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

	state.FileWalker.Position = 0
	state.FileWalker.LibSize = cfg.Local.EstimatedLibSize
	matches := make([]media.File, 0)

	libCache := &cache.Instance().Library

	// if redis is enabled the cache lifetime is determined by ttl
	if redis.Get().Handle.Running() && (time.Now().Add(-cfg.Local.Redis.CacheTtl)).After(libCache.LastUpdate) {
		_ = glg.Infof("invalidating shared cache after %s due to ttl", cfg.Local.Redis.CacheTtl)
		libCache.Valid = false
	} else if !redis.Get().Handle.Running() && (time.Now().Add(-time.Minute * 5)).After(libCache.LastUpdate) {
		_ = glg.Infof("auto invalidating local lib cache after 5 minutes")
		libCache.Valid = false
	}

	fillCache := false
	if !libCache.Valid {
		fillCache = true
		previousLength := len(libCache.Data)
		libCache.Data = make([]string, 0, previousLength)
		for idx, path := range cfg.Local.MediaPaths {
			state.FileWalker.Directory = path
			_ = glg.Infof("scanning directory (%d/%d): %s", idx+1, len(cfg.Local.MediaPaths), path)
			dir_matches, err := traverseDir(file, path, fillCache)
			if err != nil {
				return []media.File{}, err
			}
			matches = append(matches, dir_matches...)
		}
		libCache.Valid = true
		libCache.LastUpdate = time.Now()
		cfg.Local.EstimatedLibSize = state.FileWalker.Position
	} else {
		_ = glg.Infof("scanning via memcache")
		state.FileWalker.Directory = "mem cache"
		matches = append(matches, traverseMemCache(file, libCache)...)
	}
	state.FileWalker.Position = 0
	// save the config file to update the library size
	_ = config.Save()
	return matches, nil
}

func traverseMemCache(file *media.File, libCache *cache.Library) []media.File {
	matches := make([]media.File, 0)
	for _, path := range libCache.Data {
		if filepath.Base(path) == file.OutName()+config.Instance().Local.Ext {
			_ = glg.Infof("found duplicate: %s", path)
			file := &media.File{Path: path}
			matches = append(matches, *file)
		}
		if state.FileWalker.Position%1000 == 0 {
			_ = glg.Logf("current dir: %s, position: %d/%d",
				filepath.Dir(path), state.FileWalker.Position, state.FileWalker.LibSize)
		}
		state.FileWalker.Position++
	}
	return matches
}

func traverseDir(file *media.File, path string, fillCache bool) ([]media.File, error) {
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
				if fillCache {
					libCache := &cache.Instance().Library
					libCache.Data = append(libCache.Data, path)
				}
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
		return nil, err
	}
	return matches, nil
}
