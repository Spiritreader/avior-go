package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/Spiritreader/avior-go/tools"
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
		if tools.InTimeSpan(client.AvailabilityStart, client.AvailabilityEnd, time.Now()) {
			canProcessJobs = true
			sleepTime = 5
		} else {
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
		_ = glg.Errorf("next job: %s", err)
		return
	}
	if job == nil {
		_ = glg.Info("no more jobs in queue")
		return
	}
	fileInfo := new(media.FileInfo)
	fileInfo.Path = job.Path
	fileInfo.Name = job.Name
	fileInfo.Subtitle = job.Subtitle
	err = fileInfo.Update()
	if err != nil {
		resume()
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
}
