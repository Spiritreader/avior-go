package main

import (
	"bufio"
	"context"
	"os"
	"path/filepath"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
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
	dataStore := db.Get()
	database := aviorDb.Db()
	_ = dataStore.LoadSharedConfig()
	tempMany := make([]structs.Field, 0)
	tempOne := make([]structs.Field, 0)

	newField := structs.Field{ID: primitive.NilObjectID, Value: "Exclude this, you filthy casual"}
	tempOne = append(tempOne, newField)

	newField2 := &structs.Field{Value: "Exclude this, you filthy casual2"}
	tempMany = append(tempMany, *newField2)

	newField3 := &structs.Field{Value: "Exclude this, you filthy casual3"}
	tempMany = append(tempMany, *newField3)

	_ = aviorDb.InsertFields(database.Collection("name_exclude"), tempOne)
	_ = aviorDb.InsertFields(database.Collection("name_exclude"), tempMany)
	_ = aviorDb.DeleteFields(database.Collection("name_exclude"), tempOne)
	_ = aviorDb.DeleteFields(database.Collection("name_exclude"), tempMany)
}

func readFileContent(out *[]structs.Field, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return err
	}
	fileHandle, err := os.Open(filePath)
	if err != nil {
		_ = glg.Errorf("couldn't open file %s", filePath)
		return err
	}
	defer fileHandle.Close()

	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		newField := new(structs.Field)
		newField.Value = scanner.Text()
		*out = append(*out, *newField)
	}
	if err := scanner.Err(); err != nil {
		_ = glg.Errorf("couldn't read file %s", filePath)
		*out = nil
		return err
	}
	return nil
}
