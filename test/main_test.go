package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Spiritreader/avior-go/api"
	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/Spiritreader/avior-go/tools"
	"github.com/Spiritreader/avior-go/worker"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Test(t *testing.T) {
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
	// connect to database
	_ = config.LoadLocal()
	_ = config.Save()
	aviorDb, errConnect := db.Connect()
	defer func() {
		if errConnect == nil {
			if err := aviorDb.Client().Disconnect(context.TODO()); err != nil {
				_ = glg.Errorf("error disconnecting cient, %s", err)
			}
		}
	}()
	if errConnect != nil {
		_ = glg.Errorf("error connecting to database, %s", errConnect)
		return
	}

	apiStopChan := make(chan string)
	serviceStopChan := make(chan string, 1)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go api.Run(serviceStopChan, wg, apiStopChan, aviorDb)
	state := globalstate.Instance()
	state.Paused = true
	state.PauseReason = consts.PAUSE_REASON_ENCODE_ERROR
	<-serviceStopChan
	apiStopChan <- "stop"
}

func copyTest() {
	srcPath := "D:/Recording/Master and Commander.mkv"
	dstPath := "D:/Recording/Riddick_temp/Master and Commander.mkv"
	if err := tools.MoppyFile(srcPath, dstPath, false); err != nil {
		fmt.Printf("error: %s\n", err)
	}
}

func encodeTests(aviorDb *db.DataStore) {
	serviceStopChan :=  make(chan string, 1)
	defer func() { <-serviceStopChan }()
	stopChan := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	defer wg.Wait()
	go api.Run(serviceStopChan, wg, stopChan, aviorDb)

	dataStore := db.Get()
	_ = dataStore.LoadSharedConfig()
	_ = &structs.Job{
		ID:       primitive.NewObjectID(),
		Path:     "D:/Recording/Drogen Amerikas längster Krieg - Dokumentarfilm, USA, 2012, ZDF, ZDF, 104 Mi_2015-06-25-00-25-arte (AC3,deu).mkv",
		Name:     "NEUES FRANZÖSISCHES KINO Drogen",
		Subtitle: "Amerika's längster Krieg Dokumentarfilm im Ersten",
	}
	_ = &structs.Job{
		ID:       primitive.NewObjectID(),
		Path:     "D:/Recording/Neva Give üp - Der einzig wahre Japaner.mkv",
		Name:     "NEUES FRANZÖSISCHES KINO Neva Give üp",
		Subtitle: "Der einzig wahre Japaner Dokumentarfilm im Ersten",
	}
	moi, _ := aviorDb.GetClientForMachine()
	job, _ := aviorDb.GetNextJobForClient(moi)
	jobFile := media.File{Path: job.Path, Name: job.Name, Subtitle: job.Subtitle, CustomParams: job.CustomParameters}
	err := jobFile.Update()
	if err != nil {
		stopChan <- "stop"
		return
	}
	client, _ := dataStore.GetClientForMachine()
	worker.ProcessJob(dataStore, client, job, make(chan string))
	stopChan <- "stop"
}

func insertTests(aviorDb *db.DataStore) {
	dataStore := db.Get()
	//database := aviorDb.Db()
	_ = dataStore.LoadSharedConfig()
	/*tempMany := make([]structs.Field, 0)
	tempOne := make([]structs.Field, 0)

	newField := structs.Field{ID: primitive.NilObjectID, Value: "Exclude this, you filthy casual"}
	tempOne = append(tempOne, newField)

	newField2 := &structs.Field{Value: "Exclude this, you filthy casual2"}
	tempMany = append(tempMany, *newField2)

	newField3 := &structs.Field{Value: "Exclude this, you filthy casual3"}
	tempMany = append(tempMany, *newField3)
	*/
	newJob := &structs.Job{
		ID:       primitive.NewObjectID(),
		Path:     "/ibims/einspath",
		Name:     "Die unglaublichen Abenteuer des Ying-Kai Dang",
		Subtitle: "DonnerstagsKrimi im Ersten",
	}
	client, _ := aviorDb.GetClientForMachine()
	_ = aviorDb.ModifyJob(newJob, client.ID, consts.INSERT)
	/*
		_ = aviorDb.InsertFields(database.Collection("name_exclude"), tempOne)
		_ = aviorDb.InsertFields(database.Collection("name_exclude"), tempMany)
		_ = aviorDb.DeleteFields(database.Collection("name_exclude"), tempOne)
		_ = aviorDb.DeleteFields(database.Collection("name_exclude"), tempMany)
	*/
	/*
		fields := make([]structs.Field, 0)
		_ = readFileContent(&fields, "log\\namesToCut.txt")
		_ = aviorDb.InsertFields(database.Collection("name_exclude"), fields)

		fields = make([]structs.Field, 0)
		_ = readFileContent(&fields, "log\\subtitlesToCut.txt")
		_ = aviorDb.InsertFields(database.Collection("sub_exclude"), fields)

		fields = make([]structs.Field, 0)
		_ = readFileContent(&fields, "log\\searchTermsinclude.txt")
		_ = aviorDb.InsertFields(database.Collection("log_include"), fields)

		fields = make([]structs.Field, 0)
		_ = readFileContent(&fields, "log\\searchTermsexclude.txt")
		_ = aviorDb.InsertFields(database.Collection("log_exclude"), fields)
	*/
}

func readFileContent(out *[]structs.Field, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return err
	}
	fileHandle, err := os.Open(filePath)
	if err != nil {
		_ = glg.Errorf("could not open file %s", filePath)
		return err
	}
	defer fileHandle.Close()

	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		scannerText := scanner.Text()
		if strings.HasPrefix(scannerText, "#") {
			continue
		}
		newField := new(structs.Field)
		newField.Value = scannerText
		*out = append(*out, *newField)
	}
	if err := scanner.Err(); err != nil {
		_ = glg.Errorf("could not read file %s", filePath)
		*out = nil
		return err
	}
	return nil
}
