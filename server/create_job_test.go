package server

// Tests specific to the POST /v1/jobs endpoint.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shyp/rest"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/test"
)

var u = &UnsafeBypassAuthorizer{}

func Test405WrongMethod(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/jobs", nil)
	test.AssertNotError(t, err, "")
	DefaultServer.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusMethodNotAllowed)
	var e rest.Error
	err = json.Unmarshal(w.Body.Bytes(), &e)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, e.Title, "Method not allowed")
	test.AssertEquals(t, e.Instance, "/v1/jobs")
}

func Test400MissingId(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/jobs", nil)
	test.AssertNotError(t, err, "")
	req.SetBasicAuth("foo", "bar")
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusBadRequest)
	var e rest.Error
	err = json.Unmarshal(w.Body.Bytes(), &e)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, e.Title, "Missing required field: name")
	test.AssertEquals(t, e.ID, "missing_parameter")
	test.AssertEquals(t, e.Instance, "/v1/jobs")
}

func Test400MissingStrategy(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	body := CreateJobRequest{
		Name: "email-signup",
	}
	json.NewEncoder(b).Encode(body)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	test.AssertNotError(t, err, "")
	req.SetBasicAuth("foo", "bar")
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusBadRequest)
	var e rest.Error
	err = json.Unmarshal(w.Body.Bytes(), &e)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, e.Title, "Missing required field: delivery_strategy")
	test.AssertEquals(t, e.ID, "missing_parameter")
}

func Test400InvalidStrategy(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	body := CreateJobRequest{
		Name:             "email-signup",
		DeliveryStrategy: models.DeliveryStrategy("foo"),
	}
	json.NewEncoder(b).Encode(body)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	test.AssertNotError(t, err, "")
	req.SetBasicAuth("foo", "bar")
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusBadRequest)
	var e rest.Error
	err = json.Unmarshal(w.Body.Bytes(), &e)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, e.Title, "Invalid delivery strategy: foo")
	test.AssertEquals(t, e.ID, "invalid_delivery_strategy")
}

func Test400AtMostOnceAndAttempts(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	body := CreateJobRequest{
		Name:             "email-signup",
		DeliveryStrategy: models.StrategyAtMostOnce,
		Attempts:         7,
	}
	json.NewEncoder(b).Encode(body)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	test.AssertNotError(t, err, "")
	req.SetBasicAuth("foo", "bar")
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusBadRequest)
	var e rest.Error
	err = json.Unmarshal(w.Body.Bytes(), &e)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, e.Title, "Cannot set retry attempts to a number greater than 1 if the delivery strategy is at_most_once")
}

func Test400AttemptsString(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	var v interface{}
	err := json.Unmarshal([]byte(`{"name": "email-signup", "delivery_strategy": "at_most_once", "attempts": "7"}`), &v)
	test.AssertNotError(t, err, "")
	json.NewEncoder(b).Encode(v)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	req.SetBasicAuth("foo", "bar")
	test.AssertNotError(t, err, "")
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusBadRequest)
	var e rest.Error
	err = json.Unmarshal(w.Body.Bytes(), &e)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, e.Title, "Invalid request: bad JSON. Double check the types of the fields you sent")
}

func Test400ZeroAttempts(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	body := CreateJobRequest{
		Name:             "email-signup",
		DeliveryStrategy: models.StrategyAtMostOnce,
		Attempts:         0,
	}
	json.NewEncoder(b).Encode(body)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	test.AssertNotError(t, err, "")
	req.SetBasicAuth("foo", "bar")
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusBadRequest)
	var e rest.Error
	err = json.Unmarshal(w.Body.Bytes(), &e)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, e.Title, "Attempts must be set to a number greater than zero")
}

func Test400ConcurrencyNotSet(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	body := CreateJobRequest{
		Name:             "email-signup",
		DeliveryStrategy: models.StrategyAtLeastOnce,
		Attempts:         3,
	}
	json.NewEncoder(b).Encode(body)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	test.AssertNotError(t, err, "")
	req.SetBasicAuth("foo", "bar")
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusBadRequest)
	var e rest.Error
	err = json.Unmarshal(w.Body.Bytes(), &e)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, e.Title, "Concurrency must be set to a number greater than zero")
}

var validRequest = CreateJobRequest{
	Name:             "email-signup",
	DeliveryStrategy: models.StrategyAtLeastOnce,
	Attempts:         7,
	Concurrency:      3,
}

func Test401AuthorizerFailure(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(validRequest)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("usr_123", "tok_123")
	f := new(forbiddenAuthorizer)
	Get(f).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusUnauthorized)
	var e rest.Error
	err = json.Unmarshal(w.Body.Bytes(), &e)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, e.Title, "Invalid Access Token")
}

func Test401SetsToken(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(validRequest)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("usr_123", "tok_123")
	f := new(forbiddenAuthorizer)
	Get(f).ServeHTTP(w, req)
	test.AssertEquals(t, f.UserId, "usr_123")
	test.AssertEquals(t, f.Token, "tok_123")
}
