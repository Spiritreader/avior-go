package globalstate

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var once sync.Once
var instance *Data
var WaitCtxCancel context.CancelFunc
var reflectionPath string

// Instance retrieves the current configuration file instance
//
// Creates a new one if it doesn't exist
func Instance() *Data {
	once.Do(func() {
		ex, err := os.Executable()
		if err != nil {
				log.Fatal(err)
		}
		fmt.Println(ex)
		reflectionPath = filepath.Dir(ex)
		instance = new(Data)
	})
	return instance
}

func ReflectionPath() string {
	return reflectionPath
}

func (d *Data) Clear() {
	d.Encoder = Encoder{}
	d.FileWalker = FileWalker{}
	d.Mover = Mover{}
	d.InFile = ""
}

type Data struct {
	InFile          string
	Encoder         Encoder
	FileWalker      FileWalker
	Mover           Mover
	Paused          bool
	ShutdownPending bool
	HostName        string
}

type Encoder struct {
	Active            bool
	LineOut           []string `json:"-"`
	Duration          time.Time
	Frame             int
	Fps               float64
	Q                 float64
	Size              string
	Position          time.Time
	Bitrate           string
	Dup               int
	Drop              int
	Speed             float64
	Slice             int
	OfSlices          int
	Remaining         time.Duration
	Progress          float64
	ReplacementReason string
	OutPath           string
}

type FileWalker struct {
	Active    bool
	Directory string
	Position  int
	LibSize   int
}

type Mover struct {
	Active   bool
	File     string
	Progress int
	Position string
	FileSize string
}
