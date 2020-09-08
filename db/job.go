package db

import (
	"context"
	"time"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetJAllJobs gets all jobs
//
// nil will be returned if there are no jobs available
func (ds *DataStore) GetAllJobs() ([]structs.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clientCursor, err := ds.Db().Collection("jobs").Find(ctx, bson.D{})
	if err != nil {
		_ = glg.Errorf("could not retrieve jobs: %s", err)
		return nil, err
	}
	defer clientCursor.Close(ctx)
	var jobs []structs.Job
	err = clientCursor.All(ctx, &jobs)
	if err != nil {
		_ = glg.Errorf("could not read jobs: %s", err)
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, nil
	}
	return jobs, nil
}

// GetJobsForClient gets all jobs for a given client
//
// nil will be returned if there are no jobs available for the client
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
		_ = glg.Errorf("could not read jobs for client %s: %s", client.Name, err)
		return nil, err
	}
	for idx := range jobs {
		jobs[idx].AssignedClientLoaded = &client
	}
	if len(jobs) == 0 {
		return nil, nil
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
	if err != mongo.ErrNoDocuments && err != nil {
		_ = glg.Errorf("could not retrieve next job for client %s: %s", client.Name, err)
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	result.AssignedClientLoaded = client
	return result, nil
}

func (ds *DataStore) ModifyJob(job *structs.Job, clientID primitive.ObjectID, mode string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	jobColl := ds.Db().Collection("jobs")
	var err error
	switch mode {
	case consts.INSERT:
		job.ID = primitive.NewObjectID()
		job.AssignedClient = structs.DBRef{
			Ref: "clients",
			ID:  clientID,
			DB:  "undefined",
		}
		job.AssignedClientLoaded = nil
		_, err = jobColl.InsertOne(ctx, job)
	case consts.UPDATE:
		_, err = jobColl.ReplaceOne(ctx, bson.M{"_id": job.ID}, job)
	case consts.DELETE:
		_, err = jobColl.DeleteOne(ctx, bson.M{"_id": job.ID})
	}
	if err != nil {
		_ = glg.Errorf("could not %s job %s: %s", mode, job.Name, err)
		return err
	}
	_ = glg.Infof("%sd job %s", mode, job.Name)
	return nil
}
