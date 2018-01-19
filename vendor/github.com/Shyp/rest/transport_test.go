package rest

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDebugTransport(t *testing.T) {
	resp := "{\"message\": \"response\"}"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(resp))
	}))
	defer s.Close()
	c := NewClient("foo", "bar", s.URL)
	b := new(bytes.Buffer)
	c.Client.Transport = &Transport{
		Debug:  true,
		Output: b,
	}
	req, err := c.NewRequest("POST", "/request", nil)
	assertNotError(t, err, "")
	err = c.Do(req, nil)
	assertNotError(t, err, "")
	str := b.String()
	assertContains(t, str, "POST /request")
	assertContains(t, str, "200 OK")
	assertContains(t, str, resp)
}

func TestNoDebugTransport(t *testing.T) {
	resp := "{\"message\": \"response\"}"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(resp))
	}))
	defer s.Close()
	c := NewClient("foo", "bar", s.URL)
	b := new(bytes.Buffer)
	c.Client.Transport = &Transport{
		Debug:  false,
		Output: b,
	}
	req, err := c.NewRequest("POST", "/request", nil)
	assertNotError(t, err, "")
	err = c.Do(req, nil)
	assertNotError(t, err, "")
	assertEquals(t, b.Len(), 0)
}
