package downstream

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Shyp/rickover/rest"
)

const defaultHTTPTimeout = 6500 * time.Millisecond

var Logger *log.Logger

func init() {
	// setup the logger
	Logger = log.New(os.Stderr, "", log.LstdFlags)
}

var httpClient = &http.Client{Timeout: defaultHTTPTimeout}

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
	c := &Client{&rest.Client{
		Id:     id,
		Token:  token,
		Client: httpClient,
		Base:   base,
	}, nil}
	c.Job = &JobService{Client: c}
	return c
}
