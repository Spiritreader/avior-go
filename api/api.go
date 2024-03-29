package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Spiritreader/avior-go/cache"
	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/db"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kpango/glg"
)

var aviorDb *db.DataStore
var controlChan chan string

func pause(w http.ResponseWriter, r *http.Request) {
	_ = glg.Info("endpoint hit: pause service")
	state := globalstate.Instance()
	encoder := json.NewEncoder(w)
	libCache := cache.Instance()
	libCache.Library.Valid = false
	state.Paused = true
	encoder.SetIndent("", " ")
	_ = encoder.Encode("paused")
}

func resume(w http.ResponseWriter, r *http.Request) {
	_ = glg.Info("endpoint hit: resume service")
	state := globalstate.Instance()
	encoder := json.NewEncoder(w)
	state.Paused = false
	state.PauseReason = ""
	globalstate.SendWake()
	encoder.SetIndent("", " ")
	_ = encoder.Encode("resumed")
}

func getStatus(w http.ResponseWriter, r *http.Request) {
	//_ = glg.Log("endpoint hit: get status")
	state := globalstate.Instance()
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(state)
}

func getAlive(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: get alive")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode("all fine and dandy :)")
}

func getEncLineOut(w http.ResponseWriter, r *http.Request) {
	//_ = glg.Log("endpoint hit: get encoder")
	state := globalstate.Instance()
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(state.Encoder.LineOut)
}

func getLog(w http.ResponseWriter, r *http.Request, logName string) {
	content, err := os.ReadFile(filepath.Join(globalstate.ReflectionPath(), "log", logName))
	if err != nil {
		_ = glg.Errorf("could not read logfile, err: %s", err)
		_ = json.NewEncoder(w).Encode(err.Error())
		return
	}
	_, _ = w.Write(content)
}

func getMainLog(w http.ResponseWriter, r *http.Request) {
	getLog(w, r, "main.log")
}

func getErrorLog(w http.ResponseWriter, r *http.Request) {
	getLog(w, r, "err.log")
}

func getSkippedLog(w http.ResponseWriter, r *http.Request) {
	getLog(w, r, "skipped.log")
}

func getProcessedLog(w http.ResponseWriter, r *http.Request) {
	getLog(w, r, "processed.log")
}

func requestStop(w http.ResponseWriter, r *http.Request) {
	_ = glg.Info("endpoint hit: shut down service")
	select {
	case controlChan <- "stop":
		_ = glg.Info("stop signal sent to channel")
	default:
	}
	globalstate.SendWake()
	state := globalstate.Instance()
	state.ShutdownPending = true
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode("stop signal received")
}

func Run(serviceChan chan string, wg *sync.WaitGroup, apiChan chan string, db *db.DataStore) {
	controlChan = serviceChan
	aviorDb = db
	_ = glg.Infof("starting api http server")
	_ = config.LoadLocalFrom("../config.json")
	_ = config.Save()
	_ = aviorDb.LoadSharedConfig()
	cfg := config.Instance()
	port := 10000 + cfg.Local.Instance
	srv := startHttpServer(port)
	<-apiChan
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = glg.Warnf("server shutdown error, pretty harmless :), err: %s", err)
	}
	_ = glg.Infof("shutting down api http server")
	wg.Done()
}

func startHttpServer(port int) *http.Server {
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port)}

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", getStatus).Methods("GET")
	router.HandleFunc("/encoder/", getEncLineOut).Methods("GET")
	router.HandleFunc("/alive/", getAlive).Methods("GET")

	router.HandleFunc("/config/", getConfig).Methods("GET")
	router.HandleFunc("/config/", modifyConfig).Methods("PUT")

	router.HandleFunc("/fields/{id}/", getAllFields).Methods("GET")
	router.HandleFunc("/fields/{id}/", insertField).Methods("POST")
	router.HandleFunc("/fields/{id}/", updateField).Methods("PUT")
	router.HandleFunc("/fields/{id}/{el}/", deleteField).Methods("DELETE")

	router.HandleFunc("/jobs/jobsforclient/", getJobsForClient).Methods("GET")
	router.HandleFunc("/jobs/", getAllJobs).Methods("GET")
	router.HandleFunc("/jobs/", insertJob).Methods("POST")
	router.HandleFunc("/jobs/", updateJob).Methods("PUT")
	router.HandleFunc("/jobs/{id}/", deleteJob).Methods("DELETE")

	router.HandleFunc("/clients/", getAllClients).Methods("GET")
	router.HandleFunc("/clients/", insertClient).Methods("POST")
	router.HandleFunc("/clients/", updateClient).Methods("PUT")
	router.HandleFunc("/clients/{id}/", deleteClient).Methods("DELETE")

	router.HandleFunc("/shutdown/", requestStop).Methods("PUT")
	router.HandleFunc("/resume/", resume).Methods("PUT")
	router.HandleFunc("/pause/", pause).Methods("PUT")

	router.HandleFunc("/logs/main", getMainLog).Methods("GET")
	router.HandleFunc("/logs/err", getErrorLog).Methods("GET")
	router.HandleFunc("/logs/skipped", getSkippedLog).Methods("GET")
	router.HandleFunc("/logs/processed", getProcessedLog).Methods("GET")

	router.HandleFunc("/ws/status", serveWsStatus)

	/*c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})*/

	/*handlers.CORS()(router)*/
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Access-Control-Allow-Origin"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"})

	//srv.Handler = handlers.CORS(headersOk, originsOk, methodsOk)(router)
	srv.Handler = handlers.CORS(headersOk, originsOk, methodsOk)(router)
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error. port in use?
			glg.Fatalf("ListenAndServe(): %v", err)
		}
	}()
	// returning reference so caller can call Shutdown()

	return srv
}
