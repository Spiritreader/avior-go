package db

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/structs"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	testJobID1, _ := primitive.ObjectIDFromHex("5f49a61f1844fc03f4865692")
	testJobID2, _ := primitive.ObjectIDFromHex("5f49ace81844fc1a4027b9c5")
	testJobID3, _ := primitive.ObjectIDFromHex("5f49ba031844fc19c0ed54cb")
	testJob1 := &structs.Job{
		ID: testJobID1,
		Path: "\\\\UMS\\wd_usb_8tb\\Recording\\Praxis mit Meerblick - Der Prozess_2020-08-29-01-23-01-Das Erste HD (AC3,deu).ts",            
		Name: "Praxis mit Meerblick - Der Prozess",            
		Subtitle: "Spielfilm Deutschland 2018",            
		CustomParameters: nil,          
		AssignedClient: &structs.DBRef{
			Ref: "clients",
			ID:  "5ae721280b6d431584127c19",
			DB:  "undefined",
		},             
		AssignedClientLoaded: nil,           
	}
	testJob2 :=  &structs.Job{
		ID: testJobID2,
		Path: "\\\\UMS\\wd_usb_8tb\\Recording\\Uncle (1)_2020-08-29-02-48-00-Einsfestival HD (AC3,deu).ts",            
		Name: "Uncle (1)",            
		Subtitle: "Zurück auf Los",            
		CustomParameters: nil,          
		AssignedClient: &structs.DBRef{
			Ref: "clients",
			ID:  client.ID,
			DB:  "undefined",
		},             
		AssignedClientLoaded: nil,           
	}
	testJob3 :=  &structs.Job{
		ID: testJobID3,
		Path: "\\\\UMS\\wd_usb_8tb\\Recording\\Uncle (3)_2020-08-29-03-43-00-Einsfestival HD (AC3,deu).ts",            
		Name: "Uncle (3)",            
		Subtitle: "Letzter Versuch",            
		CustomParameters: nil,          
		AssignedClient: &structs.DBRef{
			Ref: "clients",
			ID:  client.ID,
			DB:  "undefined",
		},             
		AssignedClientLoaded: nil,           
	}

	// Insert job for client
	type args struct {
		job    *structs.Job
		client *structs.Client
	}
	insertJobForClientTests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: 		"InsertJobForClientTest_1",
			args{
				job: 		testJob1,
				client: client,
			},
			wantErr: 	false,
		},
		{
			name: 		"InsertJobForClientTest_2",
			args{
				job: 		testJob2,
				client: client,
			},
			wantErr: 	false,
		},
		{
			name: 		"InsertJobForClientTest_3",
			args{
				job: 		testJob3,
				client: client,
			},
			wantErr: 	false,
		},
	}
	for _, tt := range insertJobForClientTests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ds.InsertJobForClient(tt.args.job, tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.InsertJobForClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Update job
	testJob1.Path = "\\\\UMS\\wd_usb_8tb\\Recording\\Uncle (2)_2020-08-29-03-13-00-Einsfestival HD (AC3,deu).ts"
	testJob1.Name = "Uncle (2)"
	testJob1.Subtitle = "Durch dick und dünn"
	UpdateJobTests := []struct {
		name    string
		job 		*structs.Job
		wantErr bool
	}{
		{
			name: 	 "UpdateJobTest_1",
			job: 		 testJob1,
			wantErr: false
		}
	}
	for _, tt := range UpdateJobTests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ds.UpdateJob(tt.job); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.UpdateJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Get all jobs
	getAllJobsTests := []struct {
		name    string
		want    []structs.Job
		wantErr bool
	}{
		{
			name:		 "GetAllJobsTest_1",
			want:		 []structs.Job{*testJob1, *testJob2, *testJob3}
			wantErr: false,
		},
	}
	for _, tt := range getAllJobsTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ds.GetAllJobs()
			if (err != nil) != tt.wantErr {
				t.Errorf("DataStore.GetAllJobs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) == 0 {
				t.Errorf("DataStore.GetAllJobs() is returning no jobs, want some jobs")
			}
		})
	}

	// Get jobs for client
	getJobsForClientTests := []struct {
		name    string
		client 	structs.Client
		wantErr bool
	}{
		{
			"GetJobsForClientTest_1",
			client,
			false,
		},
	}
	for _, tt := range getJobsForClientTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ds.GetJobsForClient(tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataStore.GetJobsForClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) == 0 {
				t.Errorf("DataStore.GetJobsForClient() is empty, want not empty")
			}
		})
	}

	// Get next job for client
	getNextJobForClientTests := []struct {
		name    string
		client 	*structs.Client
		want    *structs.Job
		wantErr bool
	}{
		{
			name: "GetNextJobForClientTest_1",
			client,
			*testJob1,
			false,
		},
	}
	for _, tt := range getNextJobForClientTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ds.GetNextJobForClient(tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataStore.GetNextJobForClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DataStore.GetNextJobForClient() = %v, want %v", got, tt.want)
			}
		})
	}

	// Delete job
	deleteJobTests := []struct {
		name    string
		job     *structs.Job
		wantErr bool
	}{
		{
			name: 	 "DeleteTest_1",
			job: 		 &job,
			wantErr: false,
		},
	}
	for _, tt := range deleteJobTests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ds.DeleteJob(tt.job); (err != nil) != tt.wantErr {
				t.Errorf("DataStore.DeleteJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}



