package db

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// GetClientForMachine returns the current db client that matches this machine's hostname.
// A new client will be created if none is found in the database.
//
// The lookup and the insert BOTH use strings.ToUpper(hostname): the historical
// registry is uppercase (VDR-U, PHOENIX, ...), so storing the raw hostname would
// create a duplicate lower-case entry on every restart (lookup never matches).
func (ds *DataStore) GetClientForMachine() (*structs.Client, error) {
	cfg := config.Instance()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	hostname, _ := os.Hostname()
	if cfg.Local.Instance > 0 {
		hostname = fmt.Sprintf("%s-%d", hostname, cfg.Local.Instance)
	}
	hostname = strings.ToUpper(hostname)
	state := globalstate.Instance()
	state.HostName = hostname

	var thisMachine *structs.Client
	// Case-insensitive lookup: legacy entries may exist in any case (a pre-fix
	// binary stored the raw container hostname in lower case). Find them,
	// migrate their Name to UPPER case and re-save, so the registry converges
	// on a single UPPERCASE entry per machine instead of accumulating phantoms.
	collection := ds.Db().Collection("clients")
	err := collection.FindOne(ctx, bson.M{"Name": bson.M{"$regex": "^" + bsonRegexEscape(hostname) + "$", "$options": "i"}}).Decode(&thisMachine)
	if err == mongo.ErrNoDocuments {
		// Create client if it doesn't exist yet
		thisMachine = &structs.Client{
			ID:                bson.NewObjectID(),
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
	} else if thisMachine.Name != hostname {
		// Found a legacy entry in a different case: rename it to the canonical
		// UPPERCASE name. A stale lower-case twin (created by a pre-fix binary)
		// would otherwise linger forever as a phantom in the client list.
		legacyName := thisMachine.Name
		thisMachine.Name = hostname
		if err := ds.ModifyClient(thisMachine, "update"); err != nil {
			_ = glg.Warnf("could not migrate client name from %s to %s: %s", legacyName, hostname, err)
		}
		_ = glg.Infof("migrated client name %s -> %s", legacyName, hostname)
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
		return aviorClients[i].Priority < aviorClients[j].Priority
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
		client.ID = bson.NewObjectID()
		var res *mongo.InsertOneResult
		res, err = clientColl.InsertOne(ctx, client)
		client.ID = res.InsertedID.(bson.ObjectID)
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

func (ds *DataStore) DeleteClient(clientId string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clientPOID, _ := bson.ObjectIDFromHex(clientId)
	res, err := ds.Db().Collection("clients").DeleteOne(ctx, bson.M{"_id": clientPOID})
	if err != nil {
		_ = glg.Errorf("could not delete client %s: %s", clientId, err)
		return 0, err
	}
	if res.DeletedCount == 0 {
		_ = glg.Warnf("client %s to be deleted not found", clientId)
		return 0, fmt.Errorf("client %s to be deleted not found", clientId)
	} else {
		_ = glg.Infof("deleted client %s", clientId)
		return res.DeletedCount, nil
	}
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
		_ = glg.Warnf("could not sign out %s, jobs will continue to be assigned as long as it's online: %s", client.Name, err)
		return err
	}
	_ = glg.Infof("signed out %s", client.Name)
	return nil
}

func (ds *DataStore) SignOutThisClient() error {
	client, err := ds.GetClientForMachine()
	if err != nil {
		_ = glg.Warnf("could not retrieve client for current machine to sign out, jobs will continue to be assigned as long as it's online")
		return err
	}
	return ds.SignOutClient(client)
}

// bsonRegexEscape escapes regex metacharacters so a hostname can be safely
// embedded into a MongoDB $regex query (hostnames may contain dots, dashes etc.).
func bsonRegexEscape(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '.', '^', '$', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
