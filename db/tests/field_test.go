package db

import (
	"reflect"
	"testing"

	"github.com/Spiritreader/avior-go/structs"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestDataStore_SignOutClient(t *testing.T) {
	type args struct {
		client *structs.Client
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
			if err := tt.ds.SignOutClient(tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.SignOutClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_DeleteClient(t *testing.T) {
	type args struct {
		client *structs.Client
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
			if err := tt.ds.DeleteClient(tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.DeleteClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_SignInClient(t *testing.T) {
	type args struct {
		client *structs.Client
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
			if err := tt.ds.SignInClient(tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.SignInClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_UpdateClient(t *testing.T) {
	type args struct {
		client *structs.Client
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
			if err := tt.ds.UpdateClient(tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.UpdateClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_InsertClient(t *testing.T) {
	type args struct {
		client *structs.Client
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
			if err := tt.ds.InsertClient(tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.InsertClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_GetClients(t *testing.T) {
	tests := []struct {
		name    string
		ds      *DataStore
		want    []structs.Client
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ds.GetClients()
			if (err != nil) != tt.wantErr {
				t.Errorf("DataStore.GetClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DataStore.GetClients() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataStore_GetClientForMachine(t *testing.T) {
	tests := []struct {
		name    string
		ds      *DataStore
		want    *structs.Client
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ds.GetClientForMachine()
			if (err != nil) != tt.wantErr {
				t.Errorf("DataStore.GetClientForMachine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DataStore.GetClientForMachine() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
			if err := tt.ds.DeleteFields(tt.args.collection, tt.args.fields); (err != nil) != tt.wantErr {
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
			if err := tt.ds.InsertFields(tt.args.collection, tt.args.fields); (err != nil) != tt.wantErr {
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
