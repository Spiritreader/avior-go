package globalstate

import (
	"sync"
	"time"
)

var once sync.Once
var instance *Data

// Instance retrieves the current configuration file instance
//
// Creates a new one if it doesn't exist
func Instance() *Data {
	once.Do(func() {
		instance = new(Data)
	})
	return instance
}

type Data struct {
	Encoder    Encoder
	FileWalker FileWalker
}

type Encoder struct {
	LineOut   []string
	Duration  time.Time
	Frame     int
	Fps       float64
	Q         float64
	Size      string
	Position  time.Time
	Bitrate   string
	Dup       int
	Drop      int
	Speed     float64
	Slice     int
	OfSlices  int
	Remaining time.Duration
}

type FileWalker struct {
	Directory string
	Position  int
}
