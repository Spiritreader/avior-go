package cache

import (
	"sync"
	"time"
)

var once sync.Once
var instance *Data

type Data struct {
	Library Library
}

// Library Cache struct to speed up lib scan operations
type Library struct {
	Data       []string
	LastUpdate time.Time
	Valid      bool `json:"-"`
	// ScannedPaths fingerprints the MediaPaths this cache was built from. When the
	// configured MediaPaths change (config reload), the cache must be rebuilt even
	// if the TTL has not expired — otherwise files in newly added paths are never
	// seen as duplicates until a restart.
	ScannedPaths string `json:"-"`
}

// Instance retrieves the current configuration file instance
//
// Creates a new one if it doesn't exist
func Instance() *Data {
	once.Do(func() {
		instance = new(Data)
	})
	return instance
}
