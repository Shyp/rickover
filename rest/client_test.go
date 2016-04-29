package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Shyp/rickover/test"
)

func TestPost(t *testing.T) {
	t.Parallel()
	var user, pass string
	var ok bool
	var requestUrl *url.URL
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok = r.BasicAuth()
		requestUrl = r.URL
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
	}))
	defer s.Close()
	client := NewClient("foo", "bar", s.URL)
	req, err := client.NewRequest("POST", "/", nil)
	test.AssertNotError(t, err, "")
	err = client.Do(req, &struct{}{})
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, user, "foo")
	test.AssertEquals(t, pass, "bar")
	test.AssertEquals(t, requestUrl.Path, "/")
}

func TestPostError(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&Error{
			Title: "bad request",
			Id:    "something_bad",
		})
	}))
	defer s.Close()
	client := NewClient("foo", "bar", s.URL)
	req, err := client.NewRequest("POST", "/", nil)
	test.AssertNotError(t, err, "")
	err = client.Do(req, &struct{}{})
	test.AssertError(t, err, "")
	test.AssertEquals(t, err.Error(), "bad request")
}

func ExampleClient(t *testing.T) {
	client := NewClient("jobs", "secretpassword", "http://ipinfo.io")
	req, _ := client.NewRequest("GET", "/json", nil)
	type resp struct {
		City string `json:"city"`
		Ip   string `json:"ip"`
	}
	var r resp
	client.Do(req, &r)
	fmt.Println(r.Ip)
}
