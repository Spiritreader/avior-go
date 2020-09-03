package db

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/structs"
)

func TestDataStore_Job(t *testing.T) {
	_ = config.LoadLocalFrom("../config.json")
	aviorDb, errConnect := Connect()
	defer func() {
		if errConnect == nil {
			if err := aviorDb.Client().Disconnect(context.TODO()); err != nil {
				fmt.Printf("error disconnecting cient, %s", err)
			}
		}
	}()
	if errConnect != nil {
		fmt.Printf("error connecting to database, %s", errConnect)
		return
	}
	ds := Get()
	_ = ds.LoadSharedConfig()
	clients, _ := ds.GetClients()
	client := clients[0]
	

	
	deleteTests := []struct {
		name    string
		job     *structs.Job
		wantErr bool
	}{
		{
			"DeleteTest_1",
			&job,
			false,
		},
	}

	for _, tt := range deleteTests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ds.DeleteJob(tt.job); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.DeleteJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_DeleteJob(t *testing.T) {
	type args struct {
		job *structs.Job
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
			if err := tt.ds.DeleteJob(tt.args.job); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.DeleteJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_UpdateJob(t *testing.T) {
	type args struct {
		job *structs.Job
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
			if err := tt.ds.UpdateJob(tt.args.job); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.UpdateJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_InsertJobForClient(t *testing.T) {
	type args struct {
		job    *structs.Job
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
			if err := tt.ds.InsertJobForClient(tt.args.job, tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.InsertJobForClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataStore_GetNextJobForClient(t *testing.T) {
	type args struct {
		client *structs.Client
	}
	tests := []struct {
		name    string
		ds      *DataStore
		args    args
		want    *structs.Job
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ds.GetNextJobForClient(tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataStore.GetNextJobForClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DataStore.GetNextJobForClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataStore_GetJobsForClient(t *testing.T) {
	type args struct {
		client structs.Client
	}
	tests := []struct {
		name    string
		ds      *DataStore
		args    args
		want    []structs.Job
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ds.GetJobsForClient(tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataStore.GetJobsForClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DataStore.GetJobsForClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataStore_GetAllJobs(t *testing.T) {
	tests := []struct {
		name    string
		ds      *DataStore
		want    []structs.Job
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ds.GetAllJobs()
			if (err != nil) != tt.wantErr {
				t.Errorf("DataStore.GetAllJobs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DataStore.GetAllJobs() = %v, want %v", got, tt.want)
			}
		})
	}
}
