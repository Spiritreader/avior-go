package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/gorilla/mux"
	"github.com/kpango/glg"
)

var aviorDb *db.DataStore

func getStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: root")
	state := globalstate.Instance()
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(state)
}

func getAllJobs(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: getAllJobs")
	jobs, err := aviorDb.GetAllJobs()
	if err != nil {
		_ = glg.Errorf("error getting all jobs, %s", err)
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(jobs)
}

func getAllClients(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: getAllClients")
	clients, err := aviorDb.GetClients()
	if err != nil {
		_ = glg.Errorf("error getting all clients, %s", err)
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(clients)
}

func Run(ctx context.Context, wg *sync.WaitGroup) {
	_ = glg.Infof("starting api http server")
	_ = config.LoadLocalFrom("../config.json")
	_ = config.Save()
	db, errConnect := db.Connect()
	aviorDb = db
	defer func() {
		if errConnect == nil {
			if err := aviorDb.Client().Disconnect(context.TODO()); err != nil {
				if err.Error() != "client is disconnected" {
					_ = glg.Errorf("error disconnecting CIENT, %s", err)
				}
			}
		}
	}()
	if errConnect != nil {
		_ = glg.Errorf("error connecting to database, %s", errConnect)
		return
	}
	_ = aviorDb.LoadSharedConfig()
	srv := startHttpServer()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = glg.Warnf("server shutdown error, pretty harmless :), err: %s", err)
	}
	_ = glg.Infof("shutting down api http server")
	wg.Done()
}

func startHttpServer() *http.Server {
	srv := &http.Server{Addr: ":10000"}

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", getStatus)
	router.HandleFunc("/jobs", getAllJobs)
	router.HandleFunc("/clients", getAllClients)
	srv.Handler = router

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error. port in use?
			glg.Fatalf("ListenAndServe(): %v", err)
		}
	}()
	// returning reference so caller can call Shutdown()

	return srv
}
