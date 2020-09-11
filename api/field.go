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
		_ = encoder.Encode(err.Error())
		return
	}
	if mode == consts.INSERT {
		err = aviorDb.InsertFields(aviorDb.Db().Collection(keys["id"]), &fields)
		if err != nil {
			_ = glg.Errorf("could not insert fields into %s: %s", keys["id"], err)
			w.WriteHeader(http.StatusInternalServerError)
			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			_ = encoder.Encode(err.Error())
			return
		}
		w.WriteHeader(http.StatusCreated)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(fields)
		return
	} else if mode == consts.DELETE {
		var val string
		var ok bool
		if val, ok = keys["el"]; !ok {
			_ = glg.Error("hmm that shouldn't be... %s")
			w.WriteHeader(http.StatusInternalServerError)
			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			_ = encoder.Encode("key not found")
			return
		}
		err = aviorDb.DeleteField(keys["id"], val)
		if err != nil {
			_ = glg.Errorf("could not delete fields from %s: %s", keys["id"], err)
			w.WriteHeader(http.StatusInternalServerError)
			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			_ = encoder.Encode(err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(val)
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
	keys := mux.Vars(r)
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
	fields, err := aviorDb.GetFields(keys["id"])
	if err != nil {
		_ = glg.Errorf("error while retrieving %s fields, err: %s", keys["id"], err)
		w.WriteHeader(http.StatusInternalServerError)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(err.Error())
		return
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(fields)
}
