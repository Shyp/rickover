package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shyp/rest"
	"github.com/Shyp/rickover/test"
)

var jsserver = jobStatusUpdater{}

func Test400EmptyStatusBody(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	var v interface{}
	err := json.Unmarshal([]byte("{}"), &v)
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(v)
	req, _ := http.NewRequest("POST", "/v1/jobs/echo/job_123", b)
	jsserver.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusBadRequest)
	var e rest.Error
	err = json.Unmarshal(w.Body.Bytes(), &e)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, e.Title, "Missing required field: status")
	test.AssertEquals(t, e.ID, "missing_parameter")
	test.AssertEquals(t, e.Instance, "/v1/jobs/echo/job_123")
}

func Test400EmptyAttempts(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	jsr := &JobStatusRequest{
		Status:  "success",
		Attempt: nil,
	}
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(jsr)
	req, _ := http.NewRequest("POST", "/v1/jobs/echo/job_123", b)
	jsserver.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusBadRequest)
	var e rest.Error
	err := json.Unmarshal(w.Body.Bytes(), &e)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, e.Title, "Missing required field: attempt")
	test.AssertEquals(t, e.ID, "missing_parameter")
}
