package server

import (
	"net/http"

	"github.com/Shyp/rest"
)

type auther struct{}

func (a *auther) Authorize(userId, token string) *rest.Error {
	// Implement your auth scheme here.
	return nil
}

func Example() {
	// Get all server routes using your authorization handler, then listen on
	// port 9090
	handler := Get(&auther{})
	http.ListenAndServe(":9090", handler)
}
