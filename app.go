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
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	RESUME = "resume signal"
)

var (
	resumeChan chan string
	running    bool
)

func main() {
	resumeChan = make(chan string, 1)
	//Initalize global structs
	_ = config.Instance()

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
	refreshConfig()
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
		wg.Done()
		return
	}

	client, err := db.GetClientForMachine(aviorDb)
	if err != nil {
		wg.Done()
		return
	}
	err = db.SignInClient(aviorDb, client)
	if err != nil {
		_ = glg.Info("signed in %s", client.Name)
	}
	running = true
MainLoop:
	for {
		if running {
			refreshConfig()
			processJob(aviorDb, client)
		}
		select {
		case <-resumeChan:
			continue
		case <-time.After(5 * time.Minute):
			continue
		case <-ctx.Done():
			_ = glg.Info("service stop signal received")
			break MainLoop
		}
	}
	_ = db.SignOutClient(aviorDb, client)
	wg.Done()
}

func processJob(aviorDb *mongo.Database, client *db.Client) {
	job, err := db.GetNextJobForClient(aviorDb, client)
	if err != nil {
		_ = glg.Errorf("couldn't retrieve next job: %s", err)
		return
	}

	if job == nil {
		_ = glg.Info("no more jobs in queue")
		return
	}
	select {
	case resumeChan <- RESUME:
		_ = glg.Log("sending resume event")
	default:
		_ = glg.Log("resume event already waiting for consumption")
	}
}

func refreshConfig() {
	err := config.Load()
	if err != nil {
		_ = glg.Infof("could not load config: %s", err)
		return
	}
}
