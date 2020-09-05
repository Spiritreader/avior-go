package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/db"
	"github.com/gorilla/mux"
	"github.com/kpango/glg"
)

var aviorDb *db.DataStore

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the HomePage!")
	fmt.Println("Endpoint Hit: homePage")
}

func getAllJobs(w http.ResponseWriter, r *http.Request) error {
	fmt.Println("Endpoint Hit: getAllJobs")
	jobs, err := aviorDb.GetAllJobs()
	if err != nil {
		_ = glg.Errorf("error getting all jobs, %s", err)
		return err
	}
	json.NewEncoder(w).Encode(jobs)
}

func getAllClients(w http.ResponseWriter, r *http.Request) error {
	fmt.Println("Endpoint Hit: getAllClients")
	clients, err := aviorDb.GetClients()
	if err != nil {
		_ = glg.Errorf("error getting all clients, %s", err)
		return err
	}
	json.NewEncoder(w).Encode(clients)
}

func handleRequests() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homePage)
	router.HandleFunc("/alljobs", getAllJobs)
	router.HandleFunc("/allclients", getAllClients)
	_ = glg.Error(http.ListenAndServe(":10000", router))
}

func main() {
	_ = config.LoadLocalFrom("../config.json")
	_ = config.Save()
	aviorDb, errConnect := db.Connect()
	defer func() {
		if errConnect == nil {
			if err := aviorDb.Client().Disconnect(context.TODO()); err != nil {
				_ = glg.Errorf("error disconnecting cient, %s", err)
			}
		}
	}()
	if errConnect != nil {
		_ = glg.Errorf("error connecting to database, %s", errConnect)
		return
	}
	_ = aviorDb.LoadSharedConfig()
	handleRequests()
}