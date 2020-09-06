package api

import (
	"context"
	"encoding/json"
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
var appCancel context.CancelFunc

func pause(w http.ResponseWriter, r *http.Request) {
	_ = glg.Info("endpoint hit: pause service")
	state := globalstate.Instance()
	encoder := json.NewEncoder(w)
	state.Paused = true
	if globalstate.WaitCtxCancel != nil {
		globalstate.WaitCtxCancel()
	}
	encoder.SetIndent("", " ")
	_ = encoder.Encode("paused")
}

func resume(w http.ResponseWriter, r *http.Request) {
	_ = glg.Info("endpoint hit: resume service")
	state := globalstate.Instance()
	encoder := json.NewEncoder(w)
	state.Paused = false
	if globalstate.WaitCtxCancel != nil {
		globalstate.WaitCtxCancel()
	}
	encoder.SetIndent("", " ")
	_ = encoder.Encode("resumed")
}

func getStatus(w http.ResponseWriter, r *http.Request) {
	_ = glg.Info("endpoint hit: get status")
	state := globalstate.Instance()
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(state)
}

func getEncLineOut(w http.ResponseWriter, r *http.Request) {
	_ = glg.Info("endpoint hit: get encoder")
	state := globalstate.Instance()
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(state.Encoder.LineOut)
}

func requestStop(w http.ResponseWriter, r *http.Request) {
	_ = glg.Info("endpoint hit: shut down service")
	if globalstate.WaitCtxCancel != nil {
		globalstate.WaitCtxCancel()
	}
	appCancel()
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode("stop signal received")
}

func Run(cancel context.CancelFunc, wg *sync.WaitGroup, stopChan chan string, db *db.DataStore) {
	appCancel = cancel
	aviorDb = db
	_ = glg.Infof("starting api http server")
	_ = config.LoadLocalFrom("../config.json")
	_ = config.Save()
	_ = aviorDb.LoadSharedConfig()
	srv := startHttpServer()
	<-stopChan
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
	router.HandleFunc("/", getStatus).Methods("GET")
	router.HandleFunc("/encoder", getEncLineOut).Methods("GET")

	router.HandleFunc("/fields/", getAllFields).Methods("GET")
	router.HandleFunc("/fields/{id}", insertField).Methods("POST")
	router.HandleFunc("/fields/{id}", deleteField).Methods("DELETE")

	router.HandleFunc("/jobs/jobsforclient", getJobsForClient).Methods("GET")
	router.HandleFunc("/jobs/", getAllJobs).Methods("GET")
	router.HandleFunc("/jobs/", insertJob).Methods("POST")
	router.HandleFunc("/jobs/", updateJob).Methods("PUT")
	router.HandleFunc("/jobs/", deleteJob).Methods("DELETE")

	router.HandleFunc("/clients/", getAllClients).Methods("GET")
	router.HandleFunc("/clients/", insertClient).Methods("POST")
	router.HandleFunc("/clients/", updateClient).Methods("PUT")
	router.HandleFunc("/clients/", deleteClient).Methods("DELETE")

	router.HandleFunc("/shutdown", requestStop).Methods("PUT")
	router.HandleFunc("/resume", resume).Methods("PUT")
	router.HandleFunc("/pause", pause).Methods("PUT")
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
