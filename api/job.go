package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func getAllJobs(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: all jobs")
	jobs, err := aviorDb.GetAllJobs()
	if err != nil {
		_ = glg.Errorf("error getting all jobs, %s", err)
		encoder := json.NewEncoder(w)
		w.WriteHeader(http.StatusInternalServerError)
		_ = encoder.Encode(err)
		return
	}
	encoder := json.NewEncoder(w)
	w.WriteHeader(http.StatusOK)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(jobs)
}

func getJobsForClient(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: get jobs for client")
	reqBody, _ := ioutil.ReadAll(r.Body)
	var client structs.Client
	err := json.Unmarshal(reqBody, &client)
	if err != nil {
		_ = glg.Errorf("could not unmarshall client %+v: %s", string(reqBody), err)
		encoder := json.NewEncoder(w)
		w.WriteHeader(http.StatusInternalServerError)
		_ = encoder.Encode(err)
		return
	}
	jobs, err := aviorDb.GetJobsForClient(client)
	if err != nil {
		_ = glg.Errorf("error getting jobs for client %s: %s", client.Name, err)
		encoder := json.NewEncoder(w)
		w.WriteHeader(http.StatusInternalServerError)
		_ = encoder.Encode(err)
		return
	}
	encoder := json.NewEncoder(w)
	w.WriteHeader(http.StatusOK)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(jobs)
}

func insertJob(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: insert job")
	err := modifyJob(w, r, consts.INSERT)
	if err != nil {
		return
	}
}

func updateJob(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: update job")
	err := modifyJob(w, r, consts.UPDATE)
	if err != nil {
		return
	}
}

func deleteJob(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: delete job")
	err := modifyJob(w, r, consts.DELETE)
	if err != nil {
		return
	}
}

func modifyJob(w http.ResponseWriter, r *http.Request, mode string) error {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var job structs.Job
	err := json.Unmarshal(reqBody, &job)
	if err != nil {
		_ = glg.Errorf("could not unmarshall job %+v: %s", string(reqBody), err)
		w.WriteHeader(http.StatusInternalServerError)
		encoder := json.NewEncoder(w)
		_ = encoder.Encode(err)
		return err
	}
	if mode == consts.INSERT {
		var poid primitive.ObjectID
		poid, err = primitive.ObjectIDFromHex(job.AssignedClient.ID.(string))
		if err != nil {
			_ = glg.Errorf("could not %s job %s: %s", mode, job.Name, err)
		} else {
			err = aviorDb.ModifyJob(&job, poid, mode)
		}
	} else {
		// update and delete follow the same path, just the right mode has to be set.
		var poid primitive.ObjectID
		poid, err = primitive.ObjectIDFromHex(job.AssignedClient.ID.(string))
		if err != nil {
			_ = glg.Errorf("could not %s job %s: %s", mode, job.Name, err)
		} else {
			err = aviorDb.ModifyJob(&job, poid, mode)
		}
	}
	if err != nil {
		_ = glg.Errorf("could not %s job %s: %s", mode, job.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		encoder := json.NewEncoder(w)
		_ = encoder.Encode(err)
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(job)
	return nil
}
