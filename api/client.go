package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/gorilla/mux"
	"github.com/kpango/glg"
)

func getAllClients(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: all clients")
	clients, err := aviorDb.GetClients()
	if err != nil {
		_ = glg.Errorf("error getting all clients, %s", err)
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(clients)
}

func insertClient(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: insert client")
	err := modifyClient(w, r, consts.INSERT)
	if err != nil {
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func updateClient(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: update client")
	err := modifyClient(w, r, consts.UPDATE)
	if err != nil {
		return
	}
	w.WriteHeader(http.StatusOK)
}


func deleteClient(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: delete client")
	keys := mux.Vars(r)
	delAmnt, err := aviorDb.DeleteClient(keys["id"])
	if err != nil {
		if delAmnt == 0 {
			w.WriteHeader(http.StatusNotFound)
			encoder := json.NewEncoder(w)
			_ = encoder.Encode(err)
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			encoder := json.NewEncoder(w)
			_ = encoder.Encode(err)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func modifyClient(w http.ResponseWriter, r *http.Request, mode string) error {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var client *structs.Client = &structs.Client{}
	err := json.Unmarshal(reqBody, client)
	_ = glg.Logf("%s", string(reqBody))
	if err != nil {
		_ = glg.Errorf("could not unmarshal client %+v: %s", string(reqBody), err)
		w.WriteHeader(http.StatusInternalServerError)
		encoder := json.NewEncoder(w)
		_ = encoder.Encode(err)
		return err
	}
	err = aviorDb.ModifyClient(client, mode)
	if err != nil {
		_ = glg.Errorf("could not %s client %s: %s", mode, client.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		encoder := json.NewEncoder(w)
		_ = encoder.Encode(err)
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(client)
	return nil
}
