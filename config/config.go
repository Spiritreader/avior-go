package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
)

var once sync.Once
var instance *Config

// Instance retrieves the current configuration file instance
//
// Creates a new one if it doesn't exist
func Instance() *Config {
	once.Do(func() {
		instance = new(Config)
		InitWithDefaults(instance)
	})
	return instance
}

// Config is the main application configuration
type Config struct {
	DatabaseURL  string
	AudioFormats AudioFormats
	Resolutions  map[string]string
}

type AudioFormats struct {
	StereoTags []string
	MultiTags  []string
}

func InitWithDefaults(cfg *Config) {
	cfg.DatabaseURL = "mongodb://localhost:27017"
}

func Load() error {
	jsonFileHandle, err := os.Open("config.json")
	if err != nil {
		return err
	}
	defer jsonFileHandle.Close()
	serialized, err := ioutil.ReadAll(jsonFileHandle)
	if err != nil {
		return err
	}
	err = json.Unmarshal(serialized, instance)
	if err != nil {
		return err
	}
	return nil
}

func Save() error {
	encoded, err := json.MarshalIndent(*instance, "", "  ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile("config.json", encoded, 0644); err != nil {
		return err
	}
	return nil
}
