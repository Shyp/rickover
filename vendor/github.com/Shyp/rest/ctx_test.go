// +build go1.7

package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCustomBadRequest(t *testing.T) {
	err := &Error{Title: "bad"}
	RegisterHandler(400, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := CtxErr(r)
		if err == nil {
			t.Fatal("expected non-nil error, got nil")
		}
		if rerr, ok := err.(*Error); ok {
			if rerr.Title != "bad" {
				t.Errorf("expected err.Title to be 'bad', got %s", rerr.Title)
			}
		} else {
			t.Fatalf("expected err cast to Error, couldn't, err is %v", err)
		}
		w.Header().Set("Custom-Handler", "true")
		w.WriteHeader(400)
		w.Write([]byte("hello world"))
	}))
	defer func() {
		handlerMu.Lock()
		defer handlerMu.Unlock()
		delete(handlerMap, 400)
	}()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	BadRequest(w, req, err)
	if hdr := w.Header().Get("Custom-Handler"); hdr != "true" {
		t.Errorf("expected to get Custom-Handler: true header, got %s", hdr)
	}
	if body := w.Body.String(); body != "hello world" {
		t.Errorf("expected Body to be hello world, got %s", body)
	}
}
