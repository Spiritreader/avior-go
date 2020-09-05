package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kpango/glg"
)

func getAllJobs(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: all jobs")
	jobs, err := aviorDb.GetAllJobs()
	if err != nil {
		_ = glg.Errorf("error getting all jobs, %s", err)
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	_ = encoder.Encode(jobs)
}
