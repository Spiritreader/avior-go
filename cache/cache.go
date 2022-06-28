package cache

import (
	"sync"
)

var once sync.Once
var instance *Data

type Data struct {
	Library Library
}

// Library Cache struct to speed up lib scan operations
type Library struct {
	Data []string
	Valid bool
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
