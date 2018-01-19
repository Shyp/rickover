// A client for a downstream HTTP service that can do work.
//
// Once the dequeuer takes jobs out of the database, it needs to do some work
// and then report whether the work was successful or not. We do this by making
// a HTTP request to a downstream service that doesn't have any state built in,
// it just performs work when it's told to do so.
//
// This package contains a client for making requests to that downstream
// service. For example usage, check the example in this file or JobProcessor
// in the services package.
package downstream

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Shyp/rest"
)

const defaultHTTPTimeout = 6500 * time.Millisecond

var Logger *log.Logger

func init() {
	// setup the logger
	Logger = log.New(os.Stderr, "", log.LstdFlags)
}

// The DownstreamClient is an API client for a downstream service that can
// handle POST requests to /v1/jobs/:job-name/:job-id. The service is expected
// to return a 202 and then make a callback to the job scheduler when the job
// has finished running.
type Client struct {
	*rest.Client

	Job *JobService
}

// NewClient creates a new Client.
func NewClient(id, token, base string) *Client {
	rc := rest.NewClient(id, token, base)
	rc.Client = &http.Client{Timeout: defaultHTTPTimeout}
	downstreamClient := &Client{Client: rc, Job: nil}
	downstreamClient.Job = &JobService{Client: downstreamClient}
	return downstreamClient
}
