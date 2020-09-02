package main

import (
	"context"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
)

func main() {
	// connect to database
	_ = config.LoadLocal()
	aviorDb, errConnect := db.Connect()
	defer func() {
		if errConnect == nil {
			if err := aviorDb.Client.Disconnect(context.TODO()); err != nil {
				_ = glg.Errorf("error disconnecting cient, %s", err)
			}
		}
	}()
	if errConnect != nil {
		_ = glg.Errorf("error connecting to database, %s", errConnect)
		return
	}
	db.LoadSharedConfig(aviorDb.Db)
	newField := new(structs.Field)
	newField.Value = "Exclude this, you filthy casual"
	oneField := []structs.Field{*newField}
	database := aviorDb.Db
	col := database.Collection("name_exclude")
	_ = db.InsertFields(col, oneField)
}
