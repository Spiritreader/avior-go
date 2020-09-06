package globalstate

import (
	"context"
	"sync"
	"time"
)

var once sync.Once
var instance *Data
var WaitCtxCancel context.CancelFunc

// Instance retrieves the current configuration file instance
//
// Creates a new one if it doesn't exist
func Instance() *Data {
	once.Do(func() {
		instance = new(Data)
	})
	return instance
}

func (d *Data) Clear() {
	d.Encoder = Encoder{}
	d.FileWalker = FileWalker{}
	d.Mover = Mover{}
}

type Data struct {
	Encoder    Encoder
	FileWalker FileWalker
	Mover      Mover
	Paused     bool
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
