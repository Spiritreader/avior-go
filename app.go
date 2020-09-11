package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/Spiritreader/avior-go/api"
	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/Spiritreader/avior-go/tools"
	"github.com/Spiritreader/avior-go/worker"
	"github.com/kpango/glg"
	"github.com/natefinch/lumberjack"
)

var (
	resumeChan chan string
	sleep      bool
)

func main() {
	resumeChan = make(chan string, 1)
	apiChan := make(chan string)

	// Set up logger
	//log := glg.FileWriter(filepath.Join("log", "main.log"), os.ModeAppend)
	errlog := glg.FileWriter(filepath.Join("log", "err.log"), os.ModeAppend)
	log := &lumberjack.Logger{
		Filename:   filepath.Join("log", "main.log"),
		MaxSize:    10, // megabytes
		//MaxBackups: 3,
		//MaxAge:     28,   //days
		//Compress:   false, // disabled by default
	}

	glg.Get().
		SetMode(glg.BOTH).
		//AddLevelWriter(glg.LOG, log).
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

	// read cli args
	arg := os.Args[1]
	if arg == "pause" {
		globalstate.Instance().Paused = true
	}

	// Instantiate and load config file
	_ = config.Instance()
	if err := config.LoadLocal(); err != nil {
		_ = config.Save()
		_ = glg.Error("config file could not be loaded\nif this is the first startup, a new one has been created for you.\nPlease set the database url and restart the application")
		glg.Fatalf("Shutting down: %s", err)
	}

	// connect to database
	aviorDb, errConnect := db.Connect()
	defer func() {
		if errConnect == nil {
			if err := aviorDb.Client().Disconnect(context.TODO()); err != nil {
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
	wg.Add(2)
	go runService(ctx, wg, cancel, apiChan)
	go api.Run(cancel, wg, apiChan, aviorDb)
}

// runService runs the main service loop
//
// Params:
//
// ctx is the cancel context that is used to catch ctrl+c
//
// wg is the WaitGroup that is used to keep the main function waiting until
// the service exits
func runService(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, apiChan chan string) {
	state := globalstate.Instance()
	var sleepTime int
	refreshConfig()
	dataStore := db.Get()
	sleepTime = 5

	client, err := dataStore.GetClientForMachine()
	if err != nil {
		wg.Done()
		return
	}

	// sign in current machine and start loop
	err = dataStore.SignInClient(client)
	if err != nil {
		_ = glg.Info("signed in %s", client.Name)
	}
MainLoop:
	for {
		// check if client is allowed to run
		canRun := tools.InTimeSpan(client.AvailabilityStart, client.AvailabilityEnd, time.Now())
		if !canRun && !sleep {
			_ = glg.Infof("client %s moving to standby, active hours: %s - %s",
				client.Name, client.AvailabilityStart, client.AvailabilityEnd)
			sleep = false
			sleepTime = 5
		} else if canRun && sleep {
			_ = glg.Infof("client %s resuming, active hours: %s - %s",
				client.Name, client.AvailabilityStart, client.AvailabilityEnd)
			sleep = true
			sleepTime = 1
		}

		if !sleep && !state.Paused {

			refreshConfig()
			job, err := dataStore.GetNextJobForClient(client)
			if err != nil {
				_ = glg.Errorf("failed getting next job: %s", err)
			} else if job != nil {
				_, err = dataStore.DeleteJob(job.ID.Hex())
				worker.ProcessJob(dataStore, client, job, resumeChan)
				if err != nil {
					_ = glg.Failf("couldn't delete job, program has to pause to prevent endless loop")
					state.Paused = true
				}
			}
		}

		// skip sleep when more jobs are queued, also serves as exit point
		select {
		case <-ctx.Done():
			_ = glg.Info("service stop signal received")
			break MainLoop
		default:
		}
		select {
		case <-resumeChan:
			continue
		default:
		}
		// send this timer message to resume chan instead, register resumechan with the API
		// make the send non-blocking!
		waitCtx, cancel := context.WithTimeout(context.Background(), time.Duration(sleepTime)*time.Minute)
		globalstate.WaitCtxCancel = cancel
		<-waitCtx.Done()

	}
	_ = dataStore.SignOutClient(client)
	apiChan <- "stop"
	wg.Done()
	cancel()
}

func refreshConfig() {
	err := config.LoadLocal()
	if err != nil {
		_ = glg.Infof("could not load config: %s", err)
		return
	}
	err = db.Get().LoadSharedConfig()
	if err != nil {
		_ = glg.Infof("could not load shared config from db: %s", err)
	}
}
