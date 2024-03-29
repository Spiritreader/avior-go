package structs

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DATABASE
// ______________________

// Job is the Avior encode job database binding
type Job struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty"`
	Path                 string             `bson:"Path"`
	Name                 string             `bson:"Name"`
	Subtitle             string             `bson:"Subtitle"`
	CustomParameters     []string           `bson:"CustomParameters,omitempty"`
	AssignedClient       DBRef              `bson:"AssignedClient"`
	AssignedClientLoaded *Client            `bson:"AssignedClientLoaded,omitempty" json:"-"`
}

// Client is a target machine for Avior
type Client struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	Name              string             `bson:"Name"`
	AvailabilityStart string             `bson:"AvailabilityStart"`
	AvailabilityEnd   string             `bson:"AvailabilityEnd"`
	MaximumJobs       int32              `bson:"MaximumJobs"`
	Priority          int32              `bson:"Priority"`
	Online            bool               `bson:"Online"`
	IgnoreOnline      bool               `bson:"IgnoreOnline"`
}

type Field struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Value string             `bson:"Name"`
}

// DBRef wrapper to expose mongodb's references within the Go driver
type DBRef struct {
	Ref interface{} `bson:"$ref,omitempty"`
	ID  interface{} `bson:"$id"`
	DB  interface{} `bson:"$db"`
}

// CONFIG
// ______________________
