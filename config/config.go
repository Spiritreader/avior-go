package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/structs"
)

var once sync.Once
var instance *structs.Config

const (
	PRIORITY_ABOVE_NORMAL Priority = 0x00008000
	PRIORITY_BELOW_NORMAL Priority = 0x00004000
	PRIORITY_HIGH         Priority = 0x00000080
	PRIORITY_IDLE         Priority = 0x00000040
	PRIORITY_NORMAL       Priority = 0x00000020
)

type Priority uint32

func (p Priority) String() string {
	switch p {
	case PRIORITY_ABOVE_NORMAL:
		return "ABOVE_NORMAL"
	case PRIORITY_BELOW_NORMAL:
		return "BELOW_NORMAL"
	case PRIORITY_HIGH:
		return "HIGH"
	case PRIORITY_IDLE:
		return "IDLE"
	case PRIORITY_NORMAL:
		return "NORMAL"
	}
	return "UNKNOWN"
}

func PriorityUint32(p string) uint32 {
	switch p {
	case PRIORITY_ABOVE_NORMAL.String():
		return uint32(PRIORITY_ABOVE_NORMAL)
	case PRIORITY_BELOW_NORMAL.String():
		return uint32(PRIORITY_BELOW_NORMAL)
	case PRIORITY_HIGH.String():
		return uint32(PRIORITY_HIGH)
	case PRIORITY_IDLE.String():
		return uint32(PRIORITY_IDLE)
	case PRIORITY_NORMAL.String():
		return uint32(PRIORITY_NORMAL)
	}
	return uint32(PRIORITY_IDLE)
}

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
	cfg.Local.Modules = make(map[string]structs.ModuleConfig)
	cfg.Local.Resolutions = map[string]string{"hd": "1280x720", "fhd": "1920x1080"}
	cfg.Local.EncoderConfig = map[string]structs.EncoderConfig{"hd": *new(structs.EncoderConfig)}
	cfg.Local.EncoderPriority = PRIORITY_IDLE.String()

	// AgeModule Config Defaults
	moduleConfig := &structs.ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &structs.AgeModuleSettings{MaxAge: 90},
	}
	cfg.Local.Modules[consts.MODULE_NAME_AGE] = *moduleConfig
	// AudioModule Config Defaults
	moduleConfig = &structs.ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &structs.AudioModuleSettings{Accuracy: consts.AUDIO_ACC_MED},
	}
	cfg.Local.Modules[consts.MODULE_NAME_AUDIO] = *moduleConfig
	// LengthModule Config Defaults
	moduleConfig = &structs.ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &structs.LengthModuleSettings{Threshold: 25},
	}
	cfg.Local.Modules[consts.MODULE_NAME_LENGTH] = *moduleConfig
	// LogMatch Config Defaults
	moduleConfig = &structs.ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &structs.LogMatchModuleSettings{Mode: consts.LOGMATCH_MODE_NEUTRAL},
	}
	cfg.Local.Modules[consts.MODULE_NAME_LOGMATCH] = *moduleConfig
	// MaxSize Config Defaults
	moduleConfig = &structs.ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &structs.MaxSizeModuleSettings{MaxSize: 30},
	}
	// SizeApprox Config Defaults
	cfg.Local.Modules[consts.MODULE_NAME_MAXSIZE] = *moduleConfig
	moduleConfig = &structs.ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &structs.SizeApproxModuleSettings{Difference: 20, Fraction: 5, SampleCount: 2},
	}
	cfg.Local.Modules[consts.MODULE_NAME_SIZEAPPROX] = *moduleConfig
	// ResolutionModule Config Defaults
	moduleConfig = &structs.ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &structs.ResolutionModuleSettings{MinResolution: 20},
	}
	cfg.Local.Modules[consts.MODULE_NAME_SIZEAPPROX] = *moduleConfig
}

func LoadLocal() error {
	err := LoadLocalFrom("config.json")
	return err
}

func LoadLocalFrom(path string) error {
	if instance == nil {
		Instance()
	}
	jsonFileHandle, err := os.Open(path)
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
