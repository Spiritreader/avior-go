package db

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client is a target machine for Avior
type Client struct {
	ID                primitive.ObjectID `bson:"_id"`
	Name              string             `bson:"Name"`
	AvailabilityStart string             `bson:"AvailabilityStart"`
	AvailabilityEnd   string             `bson:"AvailabilityEnd"`
	Priority          int32              `bson:"Priority"`
	Online            bool               `bson:"Online"`
	IgnoreOnline      bool               `bson:"IgnoreOnline"`
}

// Job is the Avior encode job database binding
type Job struct {
	ID                   primitive.ObjectID `bson:"_id"`
	Path                 string             `bson:"Path"`
	Subtitle             string             `bson:"Subtitle"`
	AssignedClient       DBRef              `bson:"AssignedClient"`
	AssignedClientLoaded Client
}

// DBRef wrapper to expose mongodb's references within the Go driver
type DBRef struct {
	Ref interface{} `bson:"$ref"`
	ID  interface{} `bson:"$id"`
	DB  interface{} `bson:"$db"`
}

func Connect() (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg := config.Instance()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.DatabaseURL))
	if err != nil {
		return nil, err
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}
	return client.Database("Avior"), nil
}

func GetClients(db *mongo.Database) ([]Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientCursor, err := db.Collection("clients").Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer clientCursor.Close(ctx)
	var aviorClients []Client
	err = clientCursor.All(ctx, &aviorClients)
	if err != nil {
		return nil, err
	}
	sort.Slice(aviorClients, func(i, j int) bool {
		return aviorClients[i].Priority > aviorClients[j].Priority
	})
	return aviorClients, nil
}

func GetJobsForClient(db *mongo.Database, client Client) ([]Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientCursor, err := db.Collection("jobs").Find(ctx, bson.M{"AssignedClient.$id": client.ID})

	if err != nil {
		return nil, err
	}
	defer clientCursor.Close(ctx)
	var jobs []Job
	err = clientCursor.All(ctx, &jobs)
	if err != nil {
		return nil, err
	}
	for idx := range jobs {
		jobs[idx].AssignedClientLoaded = client
	}
	return jobs, nil
}

func GetNextJob(db *mongo.Database, client Client) {

}

//SayHi says hi to test modules
func SayHi() {
	fmt.Println("was good homie")
}

//SayGoodbye says goodbye to test module
func SayGoodbye() {
	fmt.Println("see ya")
}
