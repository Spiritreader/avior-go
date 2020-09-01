package db

import (
	"context"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

// Job is the Avior encode job database binding
type Job struct {
	ID                   primitive.ObjectID `bson:"_id"`
	Path                 string             `bson:"Path"`
	Name                 string             `bson:"Name"`
	Subtitle             string             `bson:"Subtitle"`
	AssignedClient       DBRef              `bson:"AssignedClient"`
	AssignedClientLoaded Client
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

var instance *DataStore
var once sync.Once

// Get establishes a connection to the database and returns the db handle
func Connect() (*DataStore, error) {
	var connectErr error
	once.Do(func() {
		instance = new(DataStore)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cfg := config.Instance()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.DatabaseURL))
		if err != nil {
			connectErr = err
			return
		}
		instance.Client = client
		instance.Db = client.Database("Avior")
	})
	if connectErr != nil {
		return nil, connectErr
	}
	return instance, nil
}

func Get() *DataStore {
	return instance
}

func GetFields(db *mongo.Database, collectionName string) ([]Field, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clientCursor, err := db.Collection(collectionName).Find(ctx, bson.D{})
	if err != nil {
		_ = glg.Errorf("couldn't retrieve all fields for collection %s: %s", collectionName, err)
		return nil, err
	}
	defer clientCursor.Close(ctx)
	var fields []Field
	err = clientCursor.All(ctx, &fields)
	if err != nil {
		return nil, err
	}
	return fields, nil
}

// GetClients retrieves all clients that have been registered
func GetClients(db *mongo.Database) ([]Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

// GetJobsForClient gets all jobs for a particular client
func GetJobsForClient(db *mongo.Database, client Client) ([]Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

// GetNextJobForClient returns the next available job in the queue for a given client
//
// nil will be returned if there are no more jobs available
func GetNextJobForClient(db *mongo.Database, client *Client) (*Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var result *Job
	err := db.Collection("jobs").FindOne(ctx, bson.M{"AssignedClient.$id": client.ID}).Decode(&result)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	result.AssignedClientLoaded = *client
	return result, nil
}

// GetClientForMachine returns the current db client that matches this machine's hostname.
// A new client will be created if none is found in the database
func GetClientForMachine(db *mongo.Database) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	hostname, _ := os.Hostname()
	var thisMachine *Client
	err := db.Collection("clients").FindOne(ctx, bson.M{"Name": strings.ToUpper(hostname)}).Decode(&thisMachine)
	if err == mongo.ErrNoDocuments {
		// Create client if it doesn't exist yet
		thisMachine = &Client{
			ID:                primitive.NewObjectID(),
			Name:              hostname,
			AvailabilityStart: "0:00",
			AvailabilityEnd:   "0:00",
			MaximumJobs:       10,
			Priority:          0,
			Online:            false,
			IgnoreOnline:      false,
		}
		err := InsertClient(db, thisMachine)
		if err != nil {
			_ = glg.Errorf("couldn't register myself as a client in the database: %s", err)
			return nil, err
		}
	} else if err != nil {
		_ = glg.Errorf("couldn't retrieve client for current machine: %s", err)
		return nil, err
	}
	return thisMachine, nil
}

func InsertClient(db *mongo.Database, client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Collection("clients").InsertOne(ctx, client)
	return err
}

func UpdateClient(db *mongo.Database, client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := db.Collection("clients").ReplaceOne(ctx, bson.M{"_id": client.ID}, client)
	if err != nil {
		_ = glg.Errorf("error updating client %s: %s", client.Name, err)
	}
	return err
}

// Signs out the current machine
func SignInClient(db *mongo.Database, client *Client) error {
	client.Online = true
	err := UpdateClient(db, client)
	if err != nil {
		_ = glg.Warnf("could not sign in %s, jobs will not be assigned to this client unless IgnoreOnline is set", client.Name)
		return err
	}
	return nil
}

// Signs out the current machine
func SignOutClient(db *mongo.Database, client *Client) error {
	client.Online = false
	err := UpdateClient(db, client)
	if err != nil {
		_ = glg.Warnf("could not sign out %s, jobs will continue to be assigned as long as its online", client.Name)
		return err
	}
	return nil
}
