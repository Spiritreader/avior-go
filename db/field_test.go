package db

import (
	"reflect"
	"testing"

	"github.com/Spiritreader/avior-go/structs"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestDataStore_DeleteFields(t *testing.T) {
	type args struct {
		collection *mongo.Collection
		fields     []structs.Field
	}
	tests := []struct {
		name    string
		ds      *DataStore
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.ds.DeleteFields(tt.args.collection, &tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.DeleteFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_InsertFields(t *testing.T) {
	type args struct {
		collection *mongo.Collection
		fields     []structs.Field
	}
	tests := []struct {
		name    string
		ds      *DataStore
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.ds.InsertFields(tt.args.collection, &tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.InsertFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_GetFields(t *testing.T) {
	type args struct {
		collectionName string
	}
	tests := []struct {
		name    string
		ds      *DataStore
		args    args
		want    []structs.Field
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ds.GetFields(tt.args.collectionName)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataStore.GetFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DataStore.GetFields() = %v, want %v", got, tt.want)
			}
		})
	}
}
