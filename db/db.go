package db

import (
	"strings"
	"sync"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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
		cfg := config.Instance()
		// driver v2 dropped the ctx arg from Connect, so the 10s bound that used to
		// come from the caller's context now lives on the client options
		client, err := mongo.Connect(options.Client().
			ApplyURI(cfg.Local.DatabaseURL).
			SetConnectTimeout(10 * time.Second))
		if err != nil {
			connectErr = err
			return
		}
		instance.client = client
		instance.db = client.Database(consts.DB_NAME)
	})
	if connectErr != nil {
		return nil, connectErr
	}
	return instance, nil
}

func (ds *DataStore) LoadSharedConfig() error {
	cfg := config.Instance()
	cfg.Shared = config.Shared{}
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

	// duplicate strip patterns (framing prefixes / module suffixes removed before
	// the duplicate comparison key is built)
	stripPatternFields, err := ds.GetFields("duplicate_strip_patterns")
	if err != nil {
		_ = glg.Errorf("could not retrieve duplicate strip patterns: %s", err)
		return err
	}
	for _, field := range stripPatternFields {
		cfg.Shared.DuplicateStripPatterns = append(cfg.Shared.DuplicateStripPatterns, field.Value)
	}

	// quality actions: one value each, defaulting to the previous behaviour
	cfg.Shared.LowerQualityAction = consts.QUALITY_ACTION_DISCARD
	if fields, err := ds.GetFields("lower_quality_action"); err != nil {
		_ = glg.Errorf("could not retrieve lower quality action: %s", err)
	} else if len(fields) > 0 {
		cfg.Shared.LowerQualityAction = strings.ToLower(strings.TrimSpace(fields[0].Value))
	}

	cfg.Shared.HigherQualityAction = consts.QUALITY_ACTION_REPLACE
	if fields, err := ds.GetFields("higher_quality_action"); err != nil {
		_ = glg.Errorf("could not retrieve higher quality action: %s", err)
	} else if len(fields) > 0 {
		cfg.Shared.HigherQualityAction = strings.ToLower(strings.TrimSpace(fields[0].Value))
	}
	return nil
}

func Get() *DataStore {
	return instance
}
