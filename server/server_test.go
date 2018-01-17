package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shyp/rickover/config"
	"github.com/Shyp/rickover/rest"
	"github.com/Shyp/rickover/test"
)

func Test404JSONUnknownResource(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/foo/unknown", nil)
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
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})
	h := forbidNonTLSTrafficHandler(mux)
	for _, tt := range prototests {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-Proto", tt.hval)
		h.ServeHTTP(w, req)
		if tt.allowed {
			test.AssertEquals(t, w.Code, 200)
		} else {
			test.AssertEquals(t, w.Code, 403)
			var e rest.Error
			err := json.Unmarshal(w.Body.Bytes(), &e)
			test.AssertNotError(t, err, "")
			test.AssertEquals(t, e.ID, "insecure_request")
		}
	}
}

func TestHomepageRendersVersion(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	req.SetBasicAuth("foo", "bar")
	u := &UnsafeBypassAuthorizer{}
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, 200)
	test.AssertEquals(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	s := w.Body.String()
	test.Assert(t, strings.Contains(s, fmt.Sprintf("rickover version %s", config.Version)), "")
}

func TestHomepageForbidsUnknownUsers(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	req.SetBasicAuth("Unknown user", "Wrong password")
	DefaultServer.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, 403)
}

func TestHomepageDisallowsUnauthedUsers(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	DefaultServer.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, 401)
}

func TestServerVersionHeader(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	req.SetBasicAuth("foo", "bar")
	u := &UnsafeBypassAuthorizer{}
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Header().Get("Server"), fmt.Sprintf("rickover/%s", config.Version))
}

func TestStrictTransportHeader(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	req.SetBasicAuth("foo", "bar")
	u := &UnsafeBypassAuthorizer{}
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Header().Get("Strict-Transport-Security"), "max-age=31536000; includeSubDomains; preload")
}
