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

func modifyFields(w http.ResponseWriter, r *http.Request, mode string) {
	keys := mux.Vars(r)
	_ = glg.Logf("endpoint hit: fields/%s", keys["id"])
	if keys["id"] != "log_exclude" &&
		keys["id"] != "log_include" &&
		keys["id"] != "name_exclude" &&
		keys["id"] != "sub_exclude" {
		_ = glg.Errorf("invalid key %s", keys["id"])
		w.WriteHeader(http.StatusInternalServerError)
		encoder := json.NewEncoder(w)
		_ = encoder.Encode("invalid name")
		return
	}
	reqBody, _ := ioutil.ReadAll(r.Body)
	var fields []structs.Field = make([]structs.Field, 0)
	err := json.Unmarshal(reqBody, &fields)
	if err != nil {
		_ = glg.Errorf("could not unmarshal field %+v: %s", string(reqBody), err)
		w.WriteHeader(http.StatusInternalServerError)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(err)
		return
	}
	if mode == consts.INSERT {
		err = aviorDb.InsertFields(aviorDb.Db().Collection(keys["id"]), &fields)
		if err != nil {
			_ = glg.Errorf("could not insert fields into %s: %s", keys["id"], err)
			w.WriteHeader(http.StatusInternalServerError)
			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			_ = encoder.Encode(err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(fields)
		return
	} else if mode == consts.DELETE {
		w.WriteHeader(http.StatusOK)
		err = aviorDb.DeleteFields(aviorDb.Db().Collection(keys["id"]), &fields)
		if err != nil {
			_ = glg.Errorf("could not delete fields from %s: %s", keys["id"], err)
			w.WriteHeader(http.StatusInternalServerError)
			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			_ = encoder.Encode(err)
			return
		}
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(fields)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(fields)
}

func insertField(w http.ResponseWriter, r *http.Request) {
	modifyFields(w, r, consts.INSERT)
}

func deleteField(w http.ResponseWriter, r *http.Request) {
	modifyFields(w, r, consts.DELETE)
}

func getAllFields(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["type"]
	if !ok || len(keys[0]) < 1 {
		_ = glg.Errorf("field request error")
		w.WriteHeader(http.StatusInternalServerError)
		encoder := json.NewEncoder(w)
		_ = encoder.Encode("field request error")
		return
	}
	var fields []structs.Field
	var err error
	if keys[0] == "sub_exclude" {
		fields, err = aviorDb.GetFields(keys[0])
	} else if keys[0] == "name_exclude" {
		fields, err = aviorDb.GetFields(keys[0])
	} else if keys[0] == "log_exclude" {
		fields, err = aviorDb.GetFields(keys[0])
	} else if keys[0] == "log_include" {
		fields, err = aviorDb.GetFields(keys[0])
	}
	if err != nil {
		_ = glg.Errorf("error while retrieving %s fields, err: %s", keys[0], err)
		w.WriteHeader(http.StatusInternalServerError)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(err)
		return
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(fields)
}
