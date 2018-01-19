// Helpers for building various types of error responses.

package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Shyp/rest"
)

func new405(r *http.Request) *rest.Error {
	return &rest.Error{
		Title:      "Method not allowed",
		ID:         "method_not_allowed",
		Instance:   r.URL.Path,
		StatusCode: 405,
	}
}

func new404(r *http.Request) *rest.Error {
	return &rest.Error{
		Title:      "Resource not found",
		ID:         "not_found",
		Instance:   r.URL.Path,
		StatusCode: 404,
	}
}

func new403(r *http.Request) *rest.Error {
	return &rest.Error{
		Title:      "Username or password are invalid. Please double check your credentials",
		ID:         "forbidden",
		Instance:   r.URL.Path,
		StatusCode: 403,
	}
}

func insecure403(r *http.Request) *rest.Error {
	return &rest.Error{
		Title:      "Server not available over HTTP",
		ID:         "insecure_request",
		Detail:     "For your security, please use an encrypted connection",
		Instance:   r.URL.Path,
		StatusCode: 403,
	}
}

func new401(r *http.Request) *rest.Error {
	return &rest.Error{
		Title:      "Unauthorized. Please include your API credentials",
		ID:         "unauthorized",
		Instance:   r.URL.Path,
		StatusCode: 401,
	}
}

// createEmptyErr returns a rest.Error indicating the request omits a required
// field.
func createEmptyErr(field string, path string) *rest.Error {
	return &rest.Error{
		Title:    fmt.Sprintf("Missing required field: %s", field),
		Detail:   fmt.Sprintf("Please include a %s in the request body", field),
		ID:       "missing_parameter",
		Instance: path,
	}
}

func createPositiveIntErr(field string, path string) *rest.Error {
	return &rest.Error{
		Title:    fmt.Sprintf("%s must be set to a number greater than zero", field),
		ID:       "invalid_parameter",
		Instance: path,
	}
}

func notFound(w http.ResponseWriter, err *rest.Error) {
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(err)
}

func badRequest(w http.ResponseWriter, r *http.Request, err *rest.Error) {
	log.Printf("400: %s %s: %s", r.Method, r.URL.Path, err.Error())
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(err)
}

func authenticate(w http.ResponseWriter, err *rest.Error) {
	w.Header().Set("WWW-Authenticate", "Basic realm=\"rickover\"")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(err)
}

func forbidden(w http.ResponseWriter, err *rest.Error) {
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(err)
}

var serverError = rest.Error{
	StatusCode: http.StatusInternalServerError,
	ID:         "server_error",
	Title:      "Unexpected server error. Please try again",
}

// writeServerError logs the provided error, and returns a generic server error
// message to the client.
func writeServerError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("500: %s %s: %s", r.Method, r.URL.Path, err)
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(serverError)
}
