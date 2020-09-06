package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/Spiritreader/avior-go/config"
	"github.com/kpango/glg"
)

func getConfig(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: get config")
	cfg := config.Instance()
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(cfg)
}

func modifyConfig(w http.ResponseWriter, r *http.Request) {
	_ = glg.Log("endpoint hit: update config")
	reqBody, _ := ioutil.ReadAll(r.Body)
	configNew := new(config.Local)
	err := json.Unmarshal(reqBody, configNew)
	if err != nil {
		_ = glg.Errorf("could not unmarshal job %+v: %s", string(reqBody), err)
		w.WriteHeader(http.StatusInternalServerError)
		encoder := json.NewEncoder(w)
		_ = encoder.Encode(err)
		return
	}
	cfg := config.Instance()
	cfg.Update(*configNew)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(config.Instance())
}
