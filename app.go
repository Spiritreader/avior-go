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
	"github.com/Spiritreader/avior-go/redis"
	"github.com/Spiritreader/avior-go/tools"
	"github.com/Spiritreader/avior-go/worker"
	"github.com/kpango/glg"
	"github.com/natefinch/lumberjack"
)

var (
	resumeChan chan string
)

func main() {
	resumeChan = make(chan string, 1)
	apiChan := make(chan string)
	serviceChan := make(chan string, 1)
	_ = globalstate.Instance()

	// Set up logger
	//log := glg.FileWriter(filepath.Join("log", "main.log"), os.ModeAppend)
	errlog := glg.FileWriter(filepath.Join(globalstate.ReflectionPath(), "log", "err.log"), os.ModeAppend)
	log := &lumberjack.Logger{
		Filename: filepath.Join(globalstate.ReflectionPath(), "log", "main.log"),
		MaxSize:  10, // megabytes
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
	_ = glg.Info("version ==>", "hey (1.5.0) codename all-g")
	defer log.Close()

	// read cli args
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "pause" {
			globalstate.Instance().Paused = true
		}
	}

	// Instantiate and load config file
	_ = config.Instance()
	if err := config.LoadLocal(); err != nil {
		copyErr := config.TryMakeCopy()
		if copyErr != nil {
			_ = glg.Errorf("error making copy of exisiting invalid config file, it may be lost, %s", copyErr)
		}
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
			_ = glg.Info("interrupt signal received, finishing all operations, please stand by")
			select {
			case serviceChan <- "stop":
				_ = glg.Info("stop signal sent to channel")
			default:
			}
			globalstate.SendWake()
			cancel()
		case <-ctx.Done():
		}
	}()

	// Run service
	wg := new(sync.WaitGroup)
	defer wg.Wait()
	wg.Add(2)
	go runService(ctx, wg, cancel, apiChan, serviceChan)
	go api.Run(serviceChan, wg, apiChan, aviorDb)
}

// runService runs the main service loop
//
// Params:
//
// ctx is the cancel context that is used to catch ctrl+c
//
// wg is the WaitGroup that is used to keep the main function waiting until
// the service exits
func runService(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, apiChan chan string, serviceChan chan string) {
	state := globalstate.Instance()
	var sleepTime int
	refreshConfig()
	dataStore := db.Get()
	sleepTime = 5

	// initial client retrieval must succeed
	client, err := dataStore.GetClientForMachine()
	if err != nil {
		_ = glg.Errorf("could not retrieve client data for machine, shutting down service")
		wg.Done()
		apiChan <- "stop"
		cancel()
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
		if !canRun && !state.Sleeping {
			_ = glg.Infof("client %s moving to standby, active hours: %s - %s",
				client.Name, client.AvailabilityStart, client.AvailabilityEnd)
			state.Sleeping = true
			sleepTime = 5
		} else if canRun && state.Sleeping {
			_ = glg.Infof("client %s resuming, active hours: %s - %s",
				client.Name, client.AvailabilityStart, client.AvailabilityEnd)
			state.Sleeping = false
			sleepTime = 1
		}

		if !state.Sleeping && !state.Paused && !state.ShutdownPending {

			refreshConfig()
			job, err := dataStore.GetNextJobForClient(client)
			if err != nil {
				_ = glg.Errorf("failed getting next job: %s", err)
			} else if job != nil {
				_, err = dataStore.DeleteJob(job.ID.Hex())
				worker.ProcessJob(dataStore, client, job, resumeChan)
				if err != nil {
					_ = glg.Failf("couldn't delete job, program has to pause to prevent it from retaking the job")
					state.Paused = true
				}
			}
		}

		// skip sleep when more jobs are queued, also serves as exit point
		select {
		case msg := <-serviceChan:
			if msg == "stop" {
				_ = glg.Info("service stop signal received")
				break MainLoop
			}
		default:
		}
		select {
		case <-resumeChan:
			continue
		default:
		}

		select {
		case <-globalstate.WakeChan():
		case <-time.After(time.Duration(sleepTime) * time.Minute):
		}

		// refresh client after sleeping time to ensure settings are updated properly
		newClient, err := dataStore.GetClientForMachine()
		if err != nil {
			_ = glg.Warnf("could not refresh client %s, using cached data", client.Name)
		} else {
			client = newClient
		}

	}
	_ = dataStore.SignOutThisClient()
	apiChan <- "stop"
	redis := redis.Get()
	redis.Handle.Close()
	wg.Done()
	cancel()
}

func refreshConfig() {
	err := config.LoadLocal()
	if err != nil {
		_ = glg.Warnf("could not refresh config: %s", err)
		time.Sleep(1 * time.Second)
		_ = glg.Warnf("retry config load")
		err = config.LoadLocal()
		if err != nil {
			_ = glg.Errorf("could not refresh config: %s", err)
			return
		}
	}
	err = db.Get().LoadSharedConfig()
	if err != nil {
		_ = glg.Infof("could not refresh shared config from db: %s", err)
	}
	redis.AutoManage()
}
