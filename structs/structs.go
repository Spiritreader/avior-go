package structs

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// DATABASE
// ______________________

// Job is the Avior encode job database binding
type Job struct {
	ID                   primitive.ObjectID `bson:"_id"`
	Path                 string             `bson:"Path"`
	Name                 string             `bson:"Name"`
	Subtitle             string             `bson:"Subtitle"`
	AssignedClient       DBRef              `bson:"AssignedClient"`
	AssignedClientLoaded Client
}

// Client is a target machine for Avior
type Client struct {
	ID                primitive.ObjectID `bson:"_id"`
	Name              string             `bson:"Name"`
	AvailabilityStart string             `bson:"AvailabilityStart"`
	AvailabilityEnd   string             `bson:"AvailabilityEnd"`
	MaximumJobs       int32              `bson:"MaximumJobs"`
	Priority          int32              `bson:"Priority"`
	Online            bool               `bson:"Online"`
	IgnoreOnline      bool               `bson:"IgnoreOnline"`
}

type Field struct {
	ID    primitive.ObjectID `bson:"_id"`
	Value string             `bson:"Name"`
}

// DBRef wrapper to expose mongodb's references within the Go driver
type DBRef struct {
	Ref interface{} `bson:"$ref"`
	ID  interface{} `bson:"$id"`
	DB  interface{} `bson:"$db"`
}

type DataStore struct {
	Db     *mongo.Database
	Client *mongo.Client
}

// CONFIG
// ______________________

type Config struct {
	Local  LocalConfig
	Shared SharedConfig
}

// LocalConfig is the main application configuration
type LocalConfig struct {
	DatabaseURL  string
	AudioFormats AudioFormats
	Resolutions  map[string]string
}

type SharedConfig struct {
	NameExclude []Field
	SubExclude  []Field
}

type AudioFormats struct {
	StereoTags []string
	MultiTags  []string
}
