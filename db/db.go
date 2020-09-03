package db

import (
	"context"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var instance *DataStore
var once sync.Once

type DataStore struct {
	db     *mongo.Database
	client *mongo.Client
}

func (ds *DataStore) Db() *mongo.Database {
	return ds.db
}

func (ds *DataStore) Client() *mongo.Client {
	return ds.client
}

// Get establishes a connection to the database and returns the db handle
func Connect() (*DataStore, error) {
	var connectErr error
	once.Do(func() {
		instance = new(DataStore)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cfg := config.Instance()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Local.DatabaseURL))
		if err != nil {
			connectErr = err
			return
		}
		instance.client = client
		instance.db = client.Database("Avior")
	})
	if connectErr != nil {
		return nil, connectErr
	}
	return instance, nil
}

func (ds *DataStore) LoadSharedConfig() error {
	cfg := config.Instance()
	nameExcludeFields, err := ds.GetFields("name_exclude")
	if err != nil {
		_ = glg.Errorf("couldn't retrieve name exclude list: %s", nameExcludeFields)
		return err
	}
	for _, field := range nameExcludeFields {
		cfg.Shared.NameExclude = append(cfg.Shared.NameExclude, field.Value)
	}
	subExcludeFields, err := ds.GetFields("sub_exclude")
	if err != nil {
		_ = glg.Errorf("couldn't retrieve sub exclude list: %s", subExcludeFields)
		return err
	}
	for _, field := range subExcludeFields {
		cfg.Shared.SubExclude = append(cfg.Shared.SubExclude, field.Value)
	}
	return nil
}

func Get() *DataStore {
	return instance
}

func (ds *DataStore) GetFields(collectionName string) ([]structs.Field, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clientCursor, err := ds.Db().Collection(collectionName).Find(ctx, bson.D{})
	if err != nil {
		_ = glg.Errorf("couldn't retrieve all fields for collection %s: %s", collectionName, err)
		return nil, err
	}
	defer clientCursor.Close(ctx)
	var fields []structs.Field
	err = clientCursor.All(ctx, &fields)
	if err != nil {
		return nil, err
	}
	return fields, nil
}

// GetJobsForClient gets all jobs for a particular client
func (ds *DataStore) GetJobsForClient(client structs.Client) ([]structs.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clientCursor, err := ds.Db().Collection("jobs").Find(ctx, bson.M{"AssignedClient.$id": client.ID})
	if err != nil {
		_ = glg.Errorf("could not retrieve jobs for client %s: %s", client.Name, err)
		return nil, err
	}
	defer clientCursor.Close(ctx)
	var jobs []structs.Job
	err = clientCursor.All(ctx, &jobs)
	if err != nil {
		_ = glg.Errorf("could not  jobs for client %s: %s", client.Name, err)
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
func (ds *DataStore) GetNextJobForClient(client *structs.Client) (*structs.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var result *structs.Job
	err := ds.Db().Collection("jobs").FindOne(ctx, bson.M{"AssignedClient.$id": client.ID}).Decode(&result)
	if err != nil {
		_ = glg.Errorf("could not retrieve next job for client %s: %s", client.Name, err)
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
func (ds *DataStore) GetClientForMachine() (*structs.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	hostname, _ := os.Hostname()
	var thisMachine *structs.Client
	err := ds.Db().Collection("clients").FindOne(ctx, bson.M{"Name": strings.ToUpper(hostname)}).Decode(&thisMachine)
	if err == mongo.ErrNoDocuments {
		// Create client if it doesn't exist yet
		thisMachine = &structs.Client{
			ID:                primitive.NewObjectID(),
			Name:              hostname,
			AvailabilityStart: "0:00",
			AvailabilityEnd:   "0:00",
			MaximumJobs:       10,
			Priority:          0,
			Online:            false,
			IgnoreOnline:      false,
		}
		err := ds.InsertClient(thisMachine)
		if err != nil {
			_ = glg.Errorf("could not register myself as a client in the database: %s", err)
			return nil, err
		}
	} else if err != nil {
		_ = glg.Errorf("could not retrieve client for current machine: %s", err)
		return nil, err
	}
	return thisMachine, nil
}

// GetClients retrieves all clients that have been registered
func (ds *DataStore) GetClients() ([]structs.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clientCursor, err := ds.Db().Collection("clients").Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer clientCursor.Close(ctx)
	var aviorClients []structs.Client
	err = clientCursor.All(ctx, &aviorClients)
	if err != nil {
		_ = glg.Errorf("could not retrieve clients: %s", err)
		return nil, err
	}
	sort.Slice(aviorClients, func(i, j int) bool {
		return aviorClients[i].Priority > aviorClients[j].Priority
	})
	return aviorClients, nil
}

func (ds *DataStore) InsertClient(client *structs.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ds.Db().Collection("clients").InsertOne(ctx, client)
	if err != nil {
		_ = glg.Errorf("could not insert client %s: %s", client.Name, err)
		return err
	}
	_ = glg.Infof("inserted client %s", client.Name)
	return nil
}

func (ds *DataStore) UpdateClient(client *structs.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := ds.Db().Collection("clients").ReplaceOne(ctx, bson.M{"_id": client.ID}, client)
	if err != nil {
		_ = glg.Errorf("could not update client %s: %s", client.Name, err)
		return err
	}
	_ = glg.Infof("updated client %s", client.Name)
	return nil
}

func (ds *DataStore) DeleteClient(client *structs.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ds.Db().Collection("clients").DeleteOne(ctx, bson.M{"_id": client.ID})
	if err != nil {
		_ = glg.Errorf("could not delete client %s: %s", client.Name, err)
		return err
	}
	_ = glg.Infof("deleted client %s", client.Name)
	return nil
}

// Signs out the current machine
func (ds *DataStore) SignInClient(client *structs.Client) error {
	client.Online = true
	err := ds.UpdateClient(client)
	if err != nil {
		_ = glg.Warnf("could not sign in %s, jobs will not be assigned to this client unless IgnoreOnline is set: %s", client.Name, err)
		return err
	}
	_ = glg.Infof("signed in %s", client.Name)
	return nil
}

// Signs out the current machine
func (ds *DataStore) SignOutClient(client *structs.Client) error {
	client.Online = false
	err := ds.UpdateClient(client)
	if err != nil {
		_ = glg.Warnf("could not sign out %s, jobs will continue to be assigned as long as its online: %s", client.Name, err)
		return err
	}
	_ = glg.Infof("signed out %s", client.Name)
	return nil
}

func (ds *DataStore) InsertFields(collection *mongo.Collection, fields []structs.Field) error {
	fieldSlice := make([]interface{}, len(fields))
	for idx, field := range fields {
		field.ID = primitive.NewObjectID()
		fieldSlice[idx] = field
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	insAmt, err := collection.InsertMany(ctx, fieldSlice)
	if err != nil {
		_ = glg.Errorf("could not insert documents into %s: %s", collection.Name(), err)
		return err
	}
	_ = glg.Infof("inserted %d documents from %s", len(insAmt.InsertedIDs), collection.Name())
	return nil
}

// DeleteFields deletes fields from the database
//
// Params:
//
// collection is the database collection to delete structs.Field structs from
//
// fields is the structs.Field slice containing all Fields that should be deleted
func (ds *DataStore) DeleteFields(collection *mongo.Collection, fields []structs.Field) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	valueSlice := make([]string, len(fields))
	for idx, field := range fields {
		valueSlice[idx] = field.Value
	}
	delAmt, err := collection.DeleteMany(ctx, bson.M{"Name": bson.M{"$in": valueSlice}})
	if err != nil {
		_ = glg.Errorf("could not delete documents from %s: %s", collection.Name(), err)
		return err
	}
	_ = glg.Infof("deleted %d documents from %s", delAmt.DeletedCount, collection.Name())
	return err
}
