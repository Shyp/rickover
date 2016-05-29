// Interface for dealing with HTTP errors.
package rest

import "fmt"

// Error implements the HTTP Problem spec laid out here:
// https://tools.ietf.org/html/draft-ietf-appsawg-http-problem-03
type Error struct {
	// The main error message. Should be short enough to fit in a phone's
	// alert box. Do not end this message with a period.
	Title string `json:"title"`

	// Id of this error message ("forbidden", "invalid_parameter", etc)
	ID string `json:"id"`

	// More information about what went wrong.
	Detail string `json:"detail,omitempty"`

	// Path to the object that's in error.
	Instance string `json:"instance,omitempty"`

	// Link to more information (Zendesk, API docs, etc)
	Type       string `json:"type,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

func (e *Error) Error() string {
	return e.Title
}

func (e *Error) String() string {
	if e.Detail != "" {
		return fmt.Sprintf("rest: %s. %s", e.Title, e.Detail)
	} else {
		return fmt.Sprintf("rest: %s", e.Title)
	}
}
