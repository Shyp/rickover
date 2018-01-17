// Package server provides an HTTP interface for the job queue/scheduler.
package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/http/pprof"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Shyp/go-dberror"
	"github.com/Shyp/go-simple-metrics"
	"github.com/Shyp/go-types"
	"github.com/Shyp/rickover/config"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/archived_jobs"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/rest"
)

// TODO(burke) use http.LimitedBytesReader.

// The maximum data size that can be sent in the body of a HTTP request.
const MAX_ENQUEUE_DATA_SIZE = 100 * 1024

var disallowUnencryptedRequests = true

// DefaultServer serves every route using the DefaultAuthorizer for
// authentication.
var DefaultServer http.Handler

// POST /v1/jobs(/:name)/:id/replay
var replayRoute = regexp.MustCompile(`^/v1/jobs(/(?P<JobName>[^\s\/]+))?/(?P<id>job_[^\s\/]+)/replay$`)

// GET /v1/jobs/job_123
//
// Must go before the getJobTypeRoute
var getJobRoute = regexp.MustCompile(`^/v1/jobs/(?P<id>job_[^\s\/]+)$`)

// GET/POST /v1/jobs
var jobsRoute = regexp.MustCompile("^/v1/jobs$")

// GET/POST/PUT /v1/jobs/:name/:id
var jobIdRoute = regexp.MustCompile(`^/v1/jobs/(?P<JobName>[^\s\/]+)/(?P<id>job_[^\s\/]+|random_id)$`)

// GET /v1/jobs/:job-name
var getJobTypeRoute = regexp.MustCompile(`^/v1/jobs/(?P<JobName>[^\s\/]+)$`)

// Get returns a http.Handler with all routes initialized using the given
// Authorizer.
func Get(a Authorizer) http.Handler {
	h := new(RegexpHandler)

	h.Handler(jobsRoute, []string{"POST"}, authHandler(createJob(), a))
	h.Handler(getJobRoute, []string{"GET"}, authHandler(handleJobRoute(), a))
	h.Handler(getJobTypeRoute, []string{"GET"}, authHandler(getJobType(), a))

	h.Handler(jobIdRoute, []string{"GET", "POST", "PUT"}, authHandler(handleJobRoute(), a))

	h.Handler(replayRoute, []string{"POST"}, authHandler(replayHandler(), a))

	h.Handler(regexp.MustCompile("^/debug/pprof$"), []string{"GET"}, authHandler(http.HandlerFunc(pprof.Index), a))
	h.Handler(regexp.MustCompile("^/debug/pprof/cmdline$"), []string{"GET"}, authHandler(http.HandlerFunc(pprof.Cmdline), a))
	h.Handler(regexp.MustCompile("^/debug/pprof/profile$"), []string{"GET"}, authHandler(http.HandlerFunc(pprof.Profile), a))
	h.Handler(regexp.MustCompile("^/debug/pprof/symbol$"), []string{"GET"}, authHandler(http.HandlerFunc(pprof.Symbol), a))
	h.Handler(regexp.MustCompile("^/debug/pprof/trace$"), []string{"GET"}, authHandler(http.HandlerFunc(pprof.Trace), a))

	h.Handler(regexp.MustCompile("^/$"), []string{"GET"}, authHandler(http.HandlerFunc(renderHomepage), a))

	return debugRequestBodyHandler(
		serverHeaderHandler(
			forbidNonTLSTrafficHandler(h),
		),
	)
}

func init() {
	DefaultServer = Get(DefaultAuthorizer)
	disallowUnencryptedRequests = os.Getenv("ALLOW_UNENCRYPTED_PROXY_TRAFFIC") != "true"
}

func serverHeaderHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// hack, figure out how to put middleware on a subset of responses
		if strings.Contains(r.URL.Path, "/debug/pprof") {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		} else if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		w.Header().Set("Server", fmt.Sprintf("rickover/%s", config.Version))
		h.ServeHTTP(w, r)
	})
}

// forbidNonTLSTrafficHandler returns a 403 to traffic that is sent via a proxy
func forbidNonTLSTrafficHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if disallowUnencryptedRequests == true {
			if r.Header.Get("X-Forwarded-Proto") == "http" {
				// It should always be set, but if it's not, let the request
				// through.
				forbidden(w, insecure403(r))
				return
			}
		}
		// This header doesn't mean anything when served over HTTP, but
		// detecting HTTPS is a general way is hard, so let's just send it
		// every time.
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		h.ServeHTTP(w, r)
	})
}

func authHandler(h http.Handler, a Authorizer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId, token, ok := r.BasicAuth()
		if !ok {
			authenticate(w, new401(r))
			return
		}
		err := a.Authorize(userId, token)
		if err != nil {
			metrics.Increment("auth.error")
			handleAuthorizeError(w, r, err)
			return
		}
		metrics.Increment("auth.success")
		h.ServeHTTP(w, r)
	})
}

// debugRequestBodyHandler prints all incoming and outgoing HTTP traffic if the
// DEBUG_HTTP_TRAFFIC environment variable is set to true. Note that the output
// will be jumbled if the server is handling multiple requests at the same
// time.
func debugRequestBodyHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("DEBUG_HTTP_TRAFFIC") == "true" {
			// You need to write the entire thing in one Write, otherwise the
			// output will be jumbled with other requests.
			b := new(bytes.Buffer)
			bits, err := httputil.DumpRequest(r, true)
			if err != nil {
				_, _ = b.WriteString(err.Error())
			} else {
				_, _ = b.Write(bits)
			}
			res := httptest.NewRecorder()
			h.ServeHTTP(res, r)

			_, _ = b.WriteString(fmt.Sprintf("HTTP/1.1 %d\r\n", res.Code))
			_ = res.HeaderMap.Write(b)
			for k, v := range res.HeaderMap {
				w.Header()[k] = v
			}
			w.WriteHeader(res.Code)
			_, _ = b.WriteString("\r\n")
			writer := io.MultiWriter(w, b)
			_, _ = res.Body.WriteTo(writer)
			_, _ = b.WriteTo(os.Stderr)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

// CreateJobRequest is a struct of data sent in the body of a request to
// /v1/jobs
type CreateJobRequest struct {
	Name             string                  `json:"name"`
	Attempts         uint8                   `json:"attempts"`
	Concurrency      uint8                   `json:"concurrency"`
	DeliveryStrategy models.DeliveryStrategy `json:"delivery_strategy"`
}

// GET /v1/jobs/:jobName
//
// Get a job type by name. Returns a models.Job or an error
func getJobType() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jobName := getJobTypeRoute.FindStringSubmatch(r.URL.Path)[1]
		job, err := jobs.Get(jobName)
		if err != nil {
			if err == sql.ErrNoRows {
				notFound(w, new404(r))
				return
			}
			writeServerError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(job)
	})
}

// POST /v1/jobs
//
// createJob returns a http.HandlerFunc that responds to job creation requests
// using the given authorizer interface.
func createJob() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			badRequest(w, r, createEmptyErr("name", r.URL.Path))
			return
		}
		defer r.Body.Close()
		var jr CreateJobRequest
		// XXX check for content-type
		err := json.NewDecoder(r.Body).Decode(&jr)
		if err != nil {
			badRequest(w, r, &rest.Error{
				ID:    "invalid_request",
				Title: "Invalid request: bad JSON. Double check the types of the fields you sent",
			})
			return
		}
		if jr.Name == "" {
			badRequest(w, r, createEmptyErr("name", r.URL.Path))
			return
		}
		if jr.DeliveryStrategy == models.DeliveryStrategy("") {
			badRequest(w, r, createEmptyErr("delivery_strategy", r.URL.Path))
			return
		}
		if jr.DeliveryStrategy != models.StrategyAtLeastOnce && jr.DeliveryStrategy != models.StrategyAtMostOnce {
			err := &rest.Error{
				Instance: r.URL.Path,
				ID:       "invalid_delivery_strategy",
				Title:    fmt.Sprintf("Invalid delivery strategy: %s", jr.DeliveryStrategy),
			}
			badRequest(w, r, err)
			return
		}

		if jr.DeliveryStrategy == models.StrategyAtMostOnce && jr.Attempts > 1 {
			err := &rest.Error{
				Instance: r.URL.Path,
				ID:       "invalid_attempts",
				Title:    "Cannot set retry attempts to a number greater than 1 if the delivery strategy is at_most_once",
				Detail:   "The at_most_once strategy implies only one attempt will be made.",
			}
			badRequest(w, r, err)
			return
		}

		if jr.Attempts == 0 {
			badRequest(w, r, createPositiveIntErr("Attempts", r.URL.Path))
			return
		}
		if jr.Concurrency == 0 {
			badRequest(w, r, createPositiveIntErr("Concurrency", r.URL.Path))
			return
		}

		jobData := models.Job{
			Name:             jr.Name,
			DeliveryStrategy: jr.DeliveryStrategy,
			Concurrency:      jr.Concurrency,
			Attempts:         jr.Attempts,
		}
		start := time.Now()
		job, err := jobs.Create(jobData)
		go metrics.Time("type.create.latency", time.Since(start))
		if err != nil {
			switch terr := err.(type) {
			case *dberror.Error:
				apierr := &rest.Error{
					Title:    terr.Message,
					ID:       "invalid_parameter",
					Instance: r.URL.Path,
				}
				badRequest(w, r, apierr)
				return
			default:
				writeServerError(w, r, err)
				return
			}
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(job)
		go metrics.Increment("type.create.success")
	})
}

// An EnqueueJobRequest is sent in the body of a request to PUT
// /v1/jobs/:job-name/:job-id.
type EnqueueJobRequest struct {
	// Job data to enqueue.
	Data json.RawMessage `json:"data"`
	// The earliest time we can run this job. If not specified, defaults to the
	// current time.
	RunAfter types.NullTime `json:"run_after"`
	// The latest time we can run this job. If not specified, defaults to null
	// (never expires).
	ExpiresAt types.NullTime `json:"expires_at"`
}

// GET/POST/PUT disambiguator for /v1/jobs/:name/:id
func handleJobRoute() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			j := jobStatusUpdater{}
			j.ServeHTTP(w, r)
		} else if r.Method == "PUT" {
			j := jobEnqueuer{}
			j.ServeHTTP(w, r)
		} else if r.Method == "GET" {
			j := jobStatusGetter{}
			j.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(new405(r))
		}
	})
}

type jobStatusGetter struct{}

// GET /v1/jobs(/:name)/:id
//
// Try to find the given job in the queued_jobs table, then in the
// archived_jobs table. Returns the job, or a 404 Not Found error.
func (j *jobStatusGetter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Job type, will be set if the longer URL form, empty string otherwise.
	var name string
	var idStr string

	// Try the longer route match first, fall back to just the ID
	jobIdMatch := jobIdRoute.FindStringSubmatch(r.URL.Path)
	if len(jobIdMatch) == 0 {
		jobIdMatch = getJobRoute.FindStringSubmatch(r.URL.Path)
		name = ""
		idStr = jobIdMatch[1]
	} else {
		name = jobIdMatch[1]
		idStr = jobIdMatch[2]
	}

	id, wroteResponse := getId(w, r, idStr)
	if wroteResponse == true {
		return
	}
	qj, err := queued_jobs.GetRetry(id, 3)
	if err == nil {
		if qj.Name != name && name != "" {
			// consider just serializing it if this is too annoying
			nfe := &rest.Error{
				Title:    "Job exists, but with a different name",
				ID:       "job_not_found",
				Instance: r.URL.Path,
			}
			notFound(w, nfe)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(qj)
		go metrics.Increment("job.get.queued.success")
		return
	}

	if err != queued_jobs.ErrNotFound {
		writeServerError(w, r, err)
		go metrics.Increment("job.get.queued.error")
		return
	}

	aj, err := archived_jobs.GetRetry(id, 3)
	if err == archived_jobs.ErrNotFound {
		notFound(w, new404(r))
		go metrics.Increment("job.get.not_found")
		return
	}
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(aj)
	go metrics.Increment("job.get.archived.success")
}

// jobEnqueuer satisfies the Handler interface.
type jobEnqueuer struct{}

// PUT /v1/jobs/:name/:id
//
// Enqueue a new job.
func (j *jobEnqueuer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		badRequest(w, r, createEmptyErr("data", r.URL.Path))
		return
	}
	defer r.Body.Close()
	var ejr EnqueueJobRequest
	err := json.NewDecoder(r.Body).Decode(&ejr)
	if err != nil {
		badRequest(w, r, &rest.Error{
			ID:    "invalid_request",
			Title: "Invalid request: bad JSON. Double check the types of the fields you sent",
		})
		return
	}
	if ejr.Data == nil {
		badRequest(w, r, createEmptyErr("data", r.URL.Path))
		return
	}
	if !ejr.RunAfter.Valid {
		ejr.RunAfter = types.NullTime{
			Valid: true,
			Time:  time.Now().UTC(),
		}
	}
	idStr := jobIdRoute.FindStringSubmatch(r.URL.Path)[2]
	var id types.PrefixUUID
	// Apache Bench can only hit one URL. This is a hack to allow random ID's
	// to be generated/inserted, even though the client is hitting the same
	// URL.
	//
	// Clients *must not* use random_id, they must generate their own UUID's.
	if idStr == "random_id" {
		id, err = types.GenerateUUID("job_")
		if err != nil {
			writeServerError(w, r, err)
			return
		}
	} else {
		var wroteResponse bool
		id, wroteResponse = getId(w, r, idStr)
		if wroteResponse == true {
			return
		}
	}
	if len(ejr.Data) > MAX_ENQUEUE_DATA_SIZE {
		err := &rest.Error{
			ID:    "entity_too_large",
			Title: "Data parameter is too large (100KB max)",
		}
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		json.NewEncoder(w).Encode(err)
		return
	}
	name := jobIdRoute.FindStringSubmatch(r.URL.Path)[1]
	queuedJob, err := queued_jobs.Enqueue(id, name, ejr.RunAfter.Time, ejr.ExpiresAt, ejr.Data)
	if err != nil {
		switch terr := err.(type) {
		case *queued_jobs.UnknownOrArchivedError:
			_, err = jobs.GetRetry(name, 3)
			if err != nil && err == sql.ErrNoRows {
				nfe := &rest.Error{
					Title:    fmt.Sprintf("Job type %s not found", name),
					ID:       "job_type_not_found",
					Instance: fmt.Sprintf("/v1/jobs/%s", name),
				}
				notFound(w, nfe)
				metrics.Increment(fmt.Sprintf("enqueue.%s.not_found", name))
				return
			} else {
				alreadyArchived := &rest.Error{
					Title:    "Job has already been archived",
					ID:       "job_already_archived",
					Instance: fmt.Sprintf("/v1/jobs/%s/%s", name, id.String()),
				}
				badRequest(w, r, alreadyArchived)
				metrics.Increment("enqueue.error.already_archived")
				return
			}
		case *dberror.Error:
			if terr.Code == dberror.CodeUniqueViolation {
				queuedJob, err = queued_jobs.Get(id)
				if err != nil {
					writeServerError(w, r, err)
					return
				}
				break
			}
			apierr := &rest.Error{
				Title:    terr.Message,
				ID:       "invalid_parameter",
				Instance: r.URL.Path,
			}
			badRequest(w, r, apierr)
			metrics.Increment(fmt.Sprintf("enqueue.%s.failure", name))
			return
		default:
			writeServerError(w, r, err)
			metrics.Increment(fmt.Sprintf("enqueue.%s.error", name))
			return
		}
	}
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(queuedJob)
	metrics.Increment(fmt.Sprintf("enqueue.success"))
	metrics.Increment(fmt.Sprintf("enqueue.%s.success", name))
}
