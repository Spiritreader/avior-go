package db

import (
	"context"
	"sync"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/kpango/glg"
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
	// name excludes
	nameExcludeFields, err := ds.GetFields("name_exclude")
	if err != nil {
		_ = glg.Errorf("could not retrieve name exclude list: %s", nameExcludeFields)
		return err
	}
	for _, field := range nameExcludeFields {
		cfg.Shared.NameExclude = append(cfg.Shared.NameExclude, field.Value)
	}

	// subtitle excludes
	subExcludeFields, err := ds.GetFields("sub_exclude")
	if err != nil {
		_ = glg.Errorf("could not retrieve sub exclude list: %s", subExcludeFields)
		return err
	}
	for _, field := range subExcludeFields {
		cfg.Shared.SubExclude = append(cfg.Shared.SubExclude, field.Value)
	}

	// log excludes
	logExcludeFields, err := ds.GetFields("log_exclude")
	if err != nil {
		_ = glg.Errorf("could not retrieve log exclude list: %s", logExcludeFields)
		return err
	}
	for _, field := range logExcludeFields {
		cfg.Shared.LogExclude = append(cfg.Shared.LogExclude, field.Value)
	}

	// log includes
	logIncludeFields, err := ds.GetFields("log_include")
	if err != nil {
		_ = glg.Errorf("could not retrieve log include list: %s", logIncludeFields)
		return err
	}
	for _, field := range logIncludeFields {
		cfg.Shared.LogInclude = append(cfg.Shared.LogInclude, field.Value)
	}
	return nil
}

func Get() *DataStore {
	return instance
}
