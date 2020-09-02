package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"

	"github.com/Spiritreader/avior-go/structs"
)

var once sync.Once
var instance *structs.Config

// Instance retrieves the current configuration file instance
//
// Creates a new one if it doesn't exist
func Instance() *structs.Config {
	once.Do(func() {
		instance = new(structs.Config)
		InitWithDefaults(instance)
	})
	return instance
}

func InitWithDefaults(cfg *structs.Config) {
	cfg.Local.DatabaseURL = "mongodb://localhost:27017"
	cfg.Local.Ext = ".mkv"
}

func LoadLocal() error {
	jsonFileHandle, err := os.Open("config.json")
	if err != nil {
		return err
	}
	defer jsonFileHandle.Close()
	serialized, err := ioutil.ReadAll(jsonFileHandle)
	if err != nil {
		return err
	}
	err = json.Unmarshal(serialized, &instance.Local)
	if err != nil {
		return err
	}
	return nil
}

func Save() error {
	encoded, err := json.MarshalIndent(instance.Local, "", "  ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile("config.json", encoded, 0644); err != nil {
		return err
	}
	return nil
}
