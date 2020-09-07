package db

import (
	"context"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetClientForMachine returns the current db client that matches this machine's hostname.
// A new client will be created if none is found in the database
func (ds *DataStore) GetClientForMachine() (*structs.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	hostname, _ := os.Hostname()
	state := globalstate.Instance()
	state.HostName = hostname
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
		err := ds.ModifyClient(thisMachine, "insert")
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

func (ds *DataStore) ModifyClient(client *structs.Client, mode string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clientColl := ds.Db().Collection("clients")
	var err error
	switch mode {
	case consts.INSERT:
		var res *mongo.InsertOneResult
		res, err = clientColl.InsertOne(ctx, client)
		client.ID = res.InsertedID.(primitive.ObjectID)
	case consts.UPDATE:
		_, err = clientColl.ReplaceOne(ctx, bson.M{"_id": client.ID}, client)
	case consts.DELETE:
		_, err = clientColl.DeleteOne(ctx, bson.M{"_id": client.ID})
	}
	if err != nil {
		_ = glg.Errorf("could not %s client %s: %s", mode, client.Name, err)
		return err
	}
	_ = glg.Infof("%sd client %s", mode, client.Name)
	return nil
}

// Signs out the current machine
func (ds *DataStore) SignInClient(client *structs.Client) error {
	client.Online = true
	err := ds.ModifyClient(client, "update")
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
	err := ds.ModifyClient(client, "update")
	if err != nil {
		_ = glg.Warnf("could not sign out %s, jobs will continue to be assigned as long as its online: %s", client.Name, err)
		return err
	}
	_ = glg.Infof("signed out %s", client.Name)
	return nil
}
