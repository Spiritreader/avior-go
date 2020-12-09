package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/globalstate"
)

var once sync.Once
var instance *Data

type Data struct {
	Local  Local
	Shared Shared
}

// Local is the main application configuration
type Local struct {
	DatabaseURL      string
	Ext              string
	AudioFormats     AudioFormats
	Resolutions      map[string]string
	ObsoletePath     string
	MediaPaths       []string
	EstimatedLibSize int
	Modules          map[string]ModuleConfig
	EncoderConfig    map[string]EncoderConfig
	EncoderPriority  string
}

type Shared struct {
	NameExclude []string
	SubExclude  []string
	LogInclude  []string
	LogExclude  []string
}

type AudioFormats struct {
	StereoTags []string
	MultiTags  []string
}

type ModuleConfig struct {
	Enabled  bool
	Priority int
	Settings interface{}
}

type AgeModuleSettings struct {
	MaxAge int
}

type AudioModuleSettings struct {
	Accuracy string
}

type LengthModuleSettings struct {
	Threshold int
}

type DuplicateLengthCheckSettings struct {
	Threshold int
}

type LogMatchModuleSettings struct {
	Mode string
}

type MaxSizeModuleSettings struct {
	MaxSize int
}

type ResolutionModuleSettings struct {
	MinResolution int
}

type SizeApproxModuleSettings struct {
	Difference  int
	SampleCount int
	Fraction    int
}

type ErrorModuleSettings struct {
	Threshold int
}

type EncoderConfig struct {
	OutDirectory  string
	PreArguments  []string
	PostArguments []string
	Stash         []string
}

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
func Instance() *Data {
	once.Do(func() {
		instance = new(Data)
		InitWithDefaults(instance)
	})
	return instance
}

func InitWithDefaults(cfg *Data) {
	cfg.Local.DatabaseURL = "mongodb://localhost:27017"
	cfg.Local.Ext = ".mkv"
	cfg.Local.Modules = make(map[string]ModuleConfig)
	cfg.Local.Resolutions = map[string]string{"hd": "1280x720", "fhd": "1920x1080"}
	cfg.Local.EncoderConfig = map[string]EncoderConfig{"hd": *new(EncoderConfig)}
	cfg.Local.EncoderPriority = PRIORITY_IDLE.String()

	// AgeModule Config Defaults
	moduleConfig := &ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &AgeModuleSettings{MaxAge: 90},
	}
	cfg.Local.Modules[consts.MODULE_NAME_AGE] = *moduleConfig
	// AudioModule Config Defaults
	moduleConfig = &ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &AudioModuleSettings{Accuracy: consts.AUDIO_ACC_MED},
	}
	cfg.Local.Modules[consts.MODULE_NAME_AUDIO] = *moduleConfig
	// LengthModule Config Defaults
	moduleConfig = &ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &LengthModuleSettings{Threshold: 25},
	}
	cfg.Local.Modules[consts.MODULE_NAME_LENGTH] = *moduleConfig
	// LogMatch Config Defaults
	moduleConfig = &ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &LogMatchModuleSettings{Mode: consts.LOGMATCH_MODE_NEUTRAL},
	}
	cfg.Local.Modules[consts.MODULE_NAME_LOGMATCH] = *moduleConfig
	// MaxSize Config Defaults
	moduleConfig = &ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &MaxSizeModuleSettings{MaxSize: 30},
	}
	// SizeApprox Config Defaults
	cfg.Local.Modules[consts.MODULE_NAME_MAXSIZE] = *moduleConfig
	moduleConfig = &ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &SizeApproxModuleSettings{Difference: 20, Fraction: 5, SampleCount: 2},
	}
	cfg.Local.Modules[consts.MODULE_NAME_SIZEAPPROX] = *moduleConfig
	// ResolutionModule Config Defaults
	moduleConfig = &ModuleConfig{
		Enabled:  false,
		Priority: 0,
		Settings: &ResolutionModuleSettings{MinResolution: 20},
	}
	cfg.Local.Modules[consts.MODULE_NAME_RESOLUTION] = *moduleConfig
	// ErrorSkipModule Config Defaults
	moduleConfig = &ModuleConfig{
		Enabled: false,
		Priority: 0,
		Settings: &ErrorModuleSettings{Threshold: 3},
	}
	cfg.Local.Modules[consts.MODULE_NAME_ERRORSKIP] = *moduleConfig
	// ErrorReplaceModule Config Defaults
	moduleConfig = &ModuleConfig{
		Enabled: false,
		Priority: 0,
		Settings: &ErrorModuleSettings{Threshold: 0},
	}
	// DuplicateLengthCheckModule Config Defaults
	cfg.Local.Modules[consts.MODULE_NAME_ERRORREPLACE] = *moduleConfig
	moduleConfig = &ModuleConfig{
		Enabled: false,
		Priority: 0,
		Settings: &DuplicateLengthCheckSettings{Threshold: 0},
	}
	cfg.Local.Modules[consts.MODULE_NAME_DUPLICATELENGTHCHECK] = *moduleConfig
}

func LoadLocal() error {
	err := LoadLocalFrom(filepath.Join(globalstate.ReflectionPath(), "config.json"))
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

func (cfg *Data) Update(inCfg Local) {
	databaseURL := cfg.Local.DatabaseURL
	cfg.Local = inCfg
	cfg.Local.DatabaseURL = databaseURL
}

func Save() error {
	encoded, err := json.MarshalIndent(instance.Local, "", "  ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(globalstate.ReflectionPath(), "config.json"), encoded, 0644); err != nil {
		return err
	}
	return nil
}
