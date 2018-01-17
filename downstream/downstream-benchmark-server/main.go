// The downstream-server is a Go server that can dequeue jobs and hit the
// callback.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
)

var accepted = []byte("{\"status\": \"accepted\"}")
var downstreamUrl string

func init() {
	downstreamUrl = os.Getenv("DOWNSTREAM_URL")
	if downstreamUrl == "" {
		downstreamUrl = "http://localhost:9090"
	}
}

type incomingRequest struct {
	Data    json.RawMessage `json:"data"`
	Attempt uint8
	Status  string
}

func successHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var ir incomingRequest
	err := json.NewDecoder(r.Body).Decode(&ir)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Bad request: %s", err.Error())))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	w.Write(accepted)
	ir.Status = "succeeded"
	go func() {
		bts, _ := json.Marshal(ir)
		req, err := http.NewRequest("POST", downstreamUrl+r.URL.Path, bytes.NewReader(bts))
		if err != nil {
			panic(err)
		}
		req.SetBasicAuth("api", os.Getenv("JOBS_SERVICE_JOBS_AUTH"))
		var resp *http.Response
		var rerr error
		for i := 0; i < 3; i++ {
			resp, rerr = http.DefaultClient.Do(req)
			if rerr != nil {
				if i == 2 {
					fmt.Println("got error:", rerr)
					break
				} else {
					continue
				}
			}
			defer resp.Body.Close()
			break
		}
		if os.Getenv("DEBUG") == "true" {
			io.Copy(os.Stdout, resp.Body)
		}
	}()
}

func main() {
	http.HandleFunc("/", successHandler)
	port := "9091"
	log.Printf("Listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port),
		handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)))

}
