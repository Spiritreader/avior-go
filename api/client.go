package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"go.mongodb.org/mongo-driver/bson"
)

func getAllClients(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: all clients")
	clients, err := aviorDb.GetClients()
	if err != nil {
		_ = glg.Errorf("error getting all clients, %s", err)
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(clients)
}

func insertClient(w http.ResponseWriter, r *http.Request) {
	modifyClient(w, r, consts.INSERT)
}

func updateClient(w http.ResponseWriter, r *http.Request) {
	modifyClient(w, r, consts.UPDATE)
}

func deleteClient(w http.ResponseWriter, r *http.Request) {
	modifyClient(w, r, consts.DELETE)
}

func modifyClient(w http.ResponseWriter, r *http.Request, method string) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var client *structs.Client
	err := bson.Unmarshal(reqBody, client)
	if err != nil {
		_ = glg.Errorf("could not unmarshall client %+v: %s", string(reqBody), err)
		return
	}
	err = aviorDb.ModifyClient(client, method)
	if err != nil {
		_ = glg.Errorf("could not %s client %s: %s", method, client.Name, err)
		return
	}
	json.NewEncoder(w).Encode(client)
}
