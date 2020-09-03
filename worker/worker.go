package worker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Spiritreader/avior-go/comparator"
	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/karrick/godirwalk"
	"github.com/kpango/glg"
)

func ProcessJob(dataStore *db.DataStore, client *structs.Client, resumeChan chan string) {
	job, err := dataStore.GetNextJobForClient(client)
	if err != nil {
		_ = glg.Errorf("failed getting next job: %s", err)
		return
	}
	if job == nil {
		return
	}
	_ = glg.Infof("processing job %s", job.Path)
	jobFile := &media.File{Path: job.Path, Name: job.Name, Subtitle: job.Subtitle, EncodeParams: job.CustomParameters}
	err = jobFile.Update()
	if err != nil {
		resume(resumeChan)
		return
	}
	fmt.Println(jobFile)
	fmt.Println(jobFile.OutName())
	runModules()
	duplicates := checkForDuplicates(jobFile)
	if dupeLen := len(duplicates); dupeLen > 0 {
		_ = glg.Infof("found %d duplicates, selecting first", dupeLen)
		runModules()
	}
	resume(resumeChan)
}

func resume(resumeChan chan string) {
	select {
	case resumeChan <- consts.RESUME:
		_ = glg.Log("sending resume event")
	default:
		_ = glg.Log("resume event already waiting for consumption")
	}
}

func runModules() {
	modules := comparator.InitDupeModules()
	for idx := range modules {
		modules[idx].Run()
	}
}

// checkForDuplicates retrieves all duplicates for the given file,
//
// given a slice of media paths that should be searched
func checkForDuplicates(file *media.File) []media.File {
	cfg := config.Instance()
	counter := 0
	matches := make([]media.File, 0)
	for idx, path := range cfg.Local.MediaPaths {
		_ = glg.Infof("scanning directory (%d/%d): %s", idx, len(cfg.Local.MediaPaths), path)
		dir_matches, count, _ := traverseDir(file, path)
		counter += count
		matches = append(matches, dir_matches...)
	}
	cfg.Local.EstimatedLibSize = counter
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
				_ = glg.Warnf("couldn't read %s, skipping: %s", path, err)
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
