package db

import (
	"context"
	"fmt"
	"time"

	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetFields gets all fields for a given collection
func (ds *DataStore) GetFields(collectionName string) ([]structs.Field, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clientCursor, err := ds.Db().Collection(collectionName).Find(ctx, bson.D{})
	if err != nil {
		_ = glg.Errorf("could not retrieve all fields for collection %s: %s", collectionName, err)
		return nil, err
	}
	defer clientCursor.Close(ctx)
	var fields []structs.Field
	err = clientCursor.All(ctx, &fields)
	if err != nil {
		_ = glg.Errorf("could not read all fields for collection %s: %s", collectionName, err)
		return nil, err
	}
	return fields, nil
}

func (ds *DataStore) InsertFields(collection *mongo.Collection, fields *[]structs.Field) error {
	fieldSlice := make([]interface{}, len(*fields))
	for idx, field := range *fields {
		field.ID = primitive.NewObjectID()
		fieldSlice[idx] = field
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	insAmt, err := collection.InsertMany(ctx, fieldSlice)
	for idx, id := range insAmt.InsertedIDs {
		(*fields)[idx].ID = id.(primitive.ObjectID)
	}
	if err != nil {
		_ = glg.Errorf("could not insert documents into %s: %s", collection.Name(), err)
		return err
	}
	_ = glg.Infof("insertd %d documents from %s", len(insAmt.InsertedIDs), collection.Name())
	return nil
}

func (ds *DataStore) UpdateFields(collectionName string, fields *[]structs.Field) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	collection := ds.Db().Collection(collectionName)
	errs := make([]error, 0)
	for idx := range *fields {
		_, err := collection.ReplaceOne(ctx, bson.M{"_id": (*fields)[idx].ID}, (*fields)[idx])
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		_ = glg.Errorf("could not update fields %s: %s", fields, errs)
		return fmt.Errorf("error updating fields: %s", errs)
	}
	_ = glg.Infof("updated fields %s", fields)
	return nil
}

func (ds *DataStore) DeleteField(collectionName string, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	poid, _ := primitive.ObjectIDFromHex(id)
	res, err := ds.Db().Collection(collectionName).DeleteOne(ctx, bson.M{"_id": poid})
	if err != nil {
		_ = glg.Errorf("could not delete field with id %s from %s", id, collectionName)
		return err
	}
	if res.DeletedCount > 0 {
		_ = glg.Infof("deleted field with id %s from %s", id, collectionName)
	} else {
		_ = glg.Infof("got id %s, but deleted nothing", id)
	}

	return nil
}

// DeleteFields deletes fields from the database
//
// Params:
//
// collection is the database collection to delete structs.Field structs from
//
// fields is the structs.Field slice containing all Fields that should be deleted
func (ds *DataStore) DeleteFields(collection *mongo.Collection, fields *[]structs.Field) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	valueSlice := make([]string, len(*fields))
	for idx, field := range *fields {
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
