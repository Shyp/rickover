package server

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/Shyp/rickover/rest"
)

var DefaultAuthorizer = NewSharedSecretAuthorizer()

// AddUser allows a given user and password to access the API.
func AddUser(user string, password string) {
	DefaultAuthorizer.AddUser(user, password)
}

// Authorizer can authorize the given user and token to access the API.
type Authorizer interface {
	Authorize(user string, token string) error
}

// SharedSecretAuthorizer uses an in-memory usernames and passwords to
// validate users.
type SharedSecretAuthorizer struct {
	allowedUsers map[string]string
	mu           sync.Mutex
}

// NewSharedSecretAuthorizer creates a SharedSecretAuthorizer ready for use.
func NewSharedSecretAuthorizer() *SharedSecretAuthorizer {
	return &SharedSecretAuthorizer{
		allowedUsers: make(map[string]string),
	}
}

// AddUser authorizes a given user and password to access the API.
func (ssa *SharedSecretAuthorizer) AddUser(userId string, password string) {
	ssa.mu.Lock()
	defer ssa.mu.Unlock()
	ssa.allowedUsers[userId] = password
}

func (c *SharedSecretAuthorizer) Authorize(userId string, token string) error {
	serverPass, ok := c.allowedUsers[userId]
	if !ok {
		if userId == "" {
			return &rest.Error{
				Title: "No authentication provided",
				Id:    "missing_authentication",
			}
		} else {
			return &rest.Error{
				Title: "Username or password are invalid. Please double check your credentials",
				Id:    "forbidden",
			}
		}
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(serverPass)) != 1 {
		return &rest.Error{
			Title: fmt.Sprintf("Incorrect password for user %s", userId),
			Id:    "incorrect_password",
		}
	}
	return nil
}

// forbiddenAuthorizer always denies access.
type forbiddenAuthorizer struct {
	UserId string
	Token  string
}

func (f *forbiddenAuthorizer) Authorize(userId string, token string) error {
	f.UserId = userId
	f.Token = token
	return &rest.Error{
		Title: "Invalid Access Token",
		Id:    "forbidden_api",
	}
}

// Use this if you need to bypass the API authorization scheme.
type UnsafeBypassAuthorizer struct{}

func (u *UnsafeBypassAuthorizer) Authorize(userId string, token string) error {
	return nil
}

// handleAuthorizeError handles a non-200 level response from the Shyp API
// (err) and writes it to the response.
func handleAuthorizeError(w http.ResponseWriter, r *http.Request, err error) {
	switch err := err.(type) {
	case *rest.Error:
		if err.Id == "forbidden_api" || err.Id == "missing_authentication" {
			err.StatusCode = 401
			authenticate(w, err)
			return
		}
		if err.Id == "incorrect_password" || err.Id == "forbidden" {
			forbidden(w, err)
			return
		}
		if err.StatusCode == http.StatusInternalServerError || err.Id == "server_error" {
			writeServerError(w, r, err)
			return
		}
		w.WriteHeader(err.StatusCode)
		json.NewEncoder(w).Encode(err)
		return
	default:
		writeServerError(w, r, err)
	}
}
