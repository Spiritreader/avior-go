package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/Spiritreader/avior-go/tools"
	"github.com/karrick/godirwalk"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	RESUME = "resume signal"
)

var (
	resumeChan     chan string
	canProcessJobs bool
)

func main() {
	resumeChan = make(chan string, 1)

	// Set up logger
	log := glg.FileWriter(filepath.Join("log", "main.log"), os.ModeAppend)
	errlog := glg.FileWriter(filepath.Join("log", "err.log"), os.ModeAppend)
	glg.Get().
		SetMode(glg.BOTH).
		AddLevelWriter(glg.LOG, log).
		AddLevelWriter(glg.INFO, log).
		AddLevelWriter(glg.WARN, log).
		AddLevelWriter(glg.DEBG, log).
		AddLevelWriter(glg.FATAL, errlog).
		AddLevelWriter(glg.ERR, errlog).
		AddLevelWriter(glg.FAIL, errlog).
		SetLevelColor(glg.ERR, glg.Red).
		SetLevelColor(glg.DEBG, glg.Cyan)
	_ = glg.Info("version ==>", "hey")
	defer log.Close()

	// Instantiate and load config file
	_ = config.Instance()
	if err := config.LoadLocal(); err != nil {
		glg.Fatalf("couldn't load config file, shutting down: %s", err)
	}

	// connect to database
	aviorDb, errConnect := db.Connect()
	defer func() {
		if errConnect == nil {
			if err := aviorDb.Client.Disconnect(context.TODO()); err != nil {
				_ = glg.Errorf("error disconnecting client, %s", err)
			}
		}
	}()
	if errConnect != nil {
		_ = glg.Errorf("error connecting to database, %s", errConnect)
		return
	}

	// Exit strategy
	ctx := context.Background()
	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()

	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	// Run service
	wg := new(sync.WaitGroup)
	defer wg.Wait()
	wg.Add(1)
	go runService(ctx, wg)
}

// runService runs the main service loop
//
// Params:
//
// ctx is the cancel context that is used to catch ctrl+c
//
// wg is the WaitGroup that is used to keep the main function waiting until
// the service exits
func runService(ctx context.Context, wg *sync.WaitGroup) {
	var sleepTime int
	refreshConfig()
	dbInstance := db.Get()

	client, err := db.GetClientForMachine(dbInstance.Db)
	if err != nil {
		wg.Done()
		return
	}

	// sign in current machine and start loop
	err = db.SignInClient(dbInstance.Db, client)
	if err != nil {
		_ = glg.Info("signed in %s", client.Name)
	}
MainLoop:
	for {
		// check if client is allowed to run
		canRun := tools.InTimeSpan(client.AvailabilityStart, client.AvailabilityEnd, time.Now())
		if canRun && !canProcessJobs {
			_ = glg.Infof("client %s moving to standby, active hours: %s - %s",
				client.Name, client.AvailabilityStart, client.AvailabilityEnd)
			canProcessJobs = true
			sleepTime = 5
		} else if !canRun && canProcessJobs {
			_ = glg.Infof("client %s resuming , active hours: %s - %s",
				client.Name, client.AvailabilityStart, client.AvailabilityEnd)
			canProcessJobs = false
			sleepTime = 1
		}

		if canProcessJobs {
			refreshConfig()
			processJob(dbInstance.Db, client)
		}

		// skip sleep when more jobs are queued, also serves as exit point
		select {
		case <-resumeChan:
			continue
		case <-time.After(time.Duration(sleepTime) * time.Minute):
			continue
		case <-ctx.Done():
			_ = glg.Info("service stop signal received")
			break MainLoop
		}
	}
	_ = db.SignOutClient(dbInstance.Db, client)
	wg.Done()
}

func processJob(aviorDb *mongo.Database, client *structs.Client) {
	job, err := db.GetNextJobForClient(aviorDb, client)
	if err != nil {
		_ = glg.Errorf("failed getting next job: %s", err)
		return
	}
	if job == nil {
		_ = glg.Info("no more jobs in queue")
		return
	}
	_ = glg.Infof("processing job %s", job.Path)
	fileInfo := new(media.FileInfo)
	fileInfo.Path = job.Path
	fileInfo.Name = job.Name
	fileInfo.Subtitle = job.Subtitle
	err = fileInfo.Update()
	if err != nil {
		resume()
	}
	fmt.Println(fileInfo)
	fmt.Println(fileInfo.OutName())
	duplicates := checkForDuplicates(fileInfo)
	if len(duplicates) > 0 {
		fmt.Println(duplicates)
	}

	resume()
}

func resume() {
	select {
	case resumeChan <- RESUME:
		_ = glg.Log("sending resume event")
	default:
		_ = glg.Log("resume event already waiting for consumption")
	}
}

func refreshConfig() {
	err := config.LoadLocal()
	if err != nil {
		_ = glg.Infof("could not load config: %s", err)
		return
	}
	err = db.LoadShared(db.Get().Db)
	if err != nil {
		_ = glg.Infof("could not load shared config from db: %s", err)
	}
}

func runModules() {
	_ = glg.Log("I am")
}

// checkForDuplicates retrieves all duplicates for the given file,
//
// given a slice of media paths that should be searched
func checkForDuplicates(file *media.FileInfo) []media.FileInfo {
	cfg := config.Instance()
	counter := 0
	matches := make([]media.FileInfo, 0)
	for _, path := range cfg.Local.MediaPaths {
		dir_matches, count, _ := traverseDir(file, path)
		counter += count
		matches = append(matches, dir_matches...)
	}
	cfg.Local.EstimatedLibSize = counter
	_ = config.Save()
	return matches
}

func traverseDir(file *media.FileInfo, path string) ([]media.FileInfo, int, error) {
	counter := 0
	matches := make([]media.FileInfo, 0)
	err := godirwalk.Walk(path, &godirwalk.Options{
		Unsorted: true,
		Callback: func(path string, de *godirwalk.Dirent) error {
			if de.IsDir() && strings.HasPrefix(de.Name(), ".") {
				_ = glg.Logf("skipping hidden dir %s", path)
				return errors.New("directory ignored")
			}
			if !de.IsDir() && strings.Contains(de.Name(), file.OutName()) {
				file := &media.FileInfo{Path: path}
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
