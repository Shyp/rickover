package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shyp/rickover/rest"
	"github.com/Shyp/rickover/test"
)

func Test404JSONUnknownResource(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo/unknown", nil)
	DefaultServer.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusNotFound)
	var e rest.Error
	err := json.Unmarshal(w.Body.Bytes(), &e)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, e.Title, "Resource not found")
	test.AssertEquals(t, e.Instance, "/foo/unknown")
}

var prototests = []struct {
	hval    string
	allowed bool
}{
	{"http", false},
	{"", true},
	{"foo", true},
	{"https", true},
}

func TestXForwardedProtoDisallowed(t *testing.T) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})
	h := forbidNonTLSTrafficHandler(http.DefaultServeMux)
	for _, tt := range prototests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-Proto", tt.hval)
		h.ServeHTTP(w, req)
		if tt.allowed {
			test.AssertEquals(t, w.Code, 200)
		} else {
			test.AssertEquals(t, w.Code, 403)
			var e rest.Error
			err := json.Unmarshal(w.Body.Bytes(), &e)
			test.AssertNotError(t, err, "")
			test.AssertEquals(t, e.Id, "insecure_request")
		}
	}
}
