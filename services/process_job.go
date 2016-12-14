package services

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-simple-metrics"
	"github.com/Shyp/rickover/downstream"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/rest"
)

// 10ms * 2^10 ~ 10 seconds between attempts
var maxMultiplier = math.Pow(2, 10)

const defaultSleepFactor = 2

// UnavailableSleepFactor determines how long the application should sleep
// between 503 Service Unavailable downstream responses.
var UnavailableSleepFactor = 500

// DefaultTimeout is the default amount of time a JobProcessor should wait for
// a job to complete, once it's been sent to the downstream server.
var DefaultTimeout = 5 * time.Minute

// JobProcessor is the default implementation of the Worker interface.
type JobProcessor struct {
	// A Client for making requests to the downstream server.
	Client *downstream.Client

	// Amount of time we should wait for the downstream server to hit the
	// callback before marking the job as failed.
	Timeout time.Duration

	// Multiplier used to determine how long to sleep between failed attempts
	// to acquire a job. The formula for sleeps is 10 * (Factor) ^ (Attempts)
	// ms. Set to 0 to not sleep between attempts.
	SleepFactor float64
}

// NewJobProcessor creates a services.JobProcessor that makes requests to the
// downstream url.
//
// By default the Client uses Basic Auth with "jobs" as the username, and the
// configured password as the password.
//
// If the downstream server does not hit the callback, jobs sent to the
// downstream server are timed out and marked as failed after DefaultTimeout
// has elapsed.
func NewJobProcessor(downstreamUrl string, downstreamPassword string) *JobProcessor {
	return &JobProcessor{
		Client:      downstream.NewClient("jobs", downstreamPassword, downstreamUrl),
		Timeout:     DefaultTimeout,
		SleepFactor: defaultSleepFactor,
	}
}

// isTimeout returns true if the err was caused by a request timeout.
func isTimeout(err error) bool {
	// This is difficult in Go 1.5: http://stackoverflow.com/a/23497404/329700
	return strings.Contains(err.Error(), "Timeout exceeded")
}

// DoWork sends the given queued job to the downstream service, then waits for
// it to complete.
func (jp *JobProcessor) DoWork(qj *models.QueuedJob) error {
	if err := jp.requestRetry(qj); err != nil {
		if isTimeout(err) {
			// Assume the request made it to Heroku; we see this most often
			// when the downstream server restarts. Heroku receives/queues the
			// requests until the new server is ready, and we see a timeout.
			return waitForJob(qj, jp.Timeout)
		} else {
			return HandleStatusCallback(qj.ID, qj.Name, models.StatusFailed, qj.Attempts, true)
		}
	}
	return waitForJob(qj, jp.Timeout)
}

// Jitter returns a value that's around the given val, but not exactly it. The
// jitter is randomly chosen between 0.8 and 1.2 times the given value, evenly
// distributed.
func jitter(val float64) float64 {
	return val*0.8 + rand.Float64()*0.2*2*val
}

func (jp JobProcessor) Sleep(failedAttempts uint32) time.Duration {
	multiplier := math.Pow(jp.SleepFactor, float64(failedAttempts))
	if multiplier > maxMultiplier {
		multiplier = maxMultiplier
	}
	return 10 * time.Duration(jitter(multiplier)) * time.Millisecond
}

func (jp *JobProcessor) requestRetry(qj *models.QueuedJob) error {
	log.Printf("processing job %s (type %s)", qj.ID.String(), qj.Name)
	for i := uint8(0); i < 3; i++ {
		if qj.ExpiresAt.Valid && time.Since(qj.ExpiresAt.Time) >= 0 {
			return createAndDelete(qj.ID, qj.Name, models.StatusExpired, qj.Attempts)
		}
		params := &downstream.JobParams{
			Data:     qj.Data,
			Attempts: qj.Attempts,
		}
		start := time.Now()
		err := jp.Client.Job.Post(qj.Name, &qj.ID, params)
		go metrics.Time("post_job.latency", time.Since(start))
		go metrics.Time(fmt.Sprintf("post_job.%s.latency", qj.Name), time.Since(start))
		if err == nil {
			go metrics.Increment(fmt.Sprintf("post_job.%s.accepted", qj.Name))
			return nil
		} else {
			switch aerr := err.(type) {
			case *rest.Error:
				if aerr.ID == "service_unavailable" {
					go metrics.Increment("post_job.unavailable")
					time.Sleep(time.Duration(1<<i*UnavailableSleepFactor) * time.Millisecond)
					continue
				}
				go metrics.Increment("dequeue.post_job.error")
				return err
			default:
				go func(err error) {
					if isTimeout(err) {
						metrics.Increment("dequeue.post_job.timeout")
					} else {
						log.Printf("Unknown error making POST request to downstream server: %#v", err)
						metrics.Increment("dequeue.post_job.error_unknown")
					}
				}(err)
				return err
			}
		}
	}
	return nil
}

func waitForJob(qj *models.QueuedJob, failTimeout time.Duration) error {
	start := time.Now()
	// This is not going to change but we continually overwrite qj
	name := qj.Name
	idStr := qj.ID.String()

	currentAttemptCount := qj.Attempts
	queryCount := int64(0)
	if failTimeout <= 0 {
		failTimeout = DefaultTimeout
	}
	timeoutChan := time.After(failTimeout)
	for {
		select {
		case <-timeoutChan:
			go metrics.Increment(fmt.Sprintf("wait_for_job.%s.timeout", name))
			log.Printf("5 minutes elapsed, marking %s (type %s) as failed", idStr, name)
			err := HandleStatusCallback(qj.ID, name, models.StatusFailed, currentAttemptCount, true)
			go metrics.Increment(fmt.Sprintf("wait_for_job.%s.failed", name))
			log.Printf("job %s (type %s) timed out after %v", idStr, name, time.Since(start))
			if err == sql.ErrNoRows {
				// Attempted to decrement the failed count, but couldn't do so;
				// we assume another thread got here before we did.
				return nil
			}
			if err != nil {
				log.Printf("error marking job %s as failed: %s\n", idStr, err.Error())
				go metrics.Increment(fmt.Sprintf("wait_for_job.%s.failed.error", name))
			}
			return err
		default:
			getStart := time.Now()
			qj, err := queued_jobs.Get(qj.ID)
			queryCount++
			go metrics.Time("wait_for_job.get.latency", time.Since(getStart))
			if err == queued_jobs.ErrNotFound {
				// inserted this job into archived_jobs. nothing to do!
				go func(name string, start time.Time, idStr string, queryCount int64) {
					metrics.Increment(fmt.Sprintf("wait_for_job.%s.archived", name))
					metrics.Increment("wait_for_job.archived")
					metrics.Time(fmt.Sprintf("wait_for_job.%s.latency", name), time.Since(start))
					metrics.Measure(fmt.Sprintf("wait_for_job.%s.queries", name), queryCount)
					duration := time.Since(start)
					// Default print method has too many decimals
					roundDuration := duration - duration%(time.Millisecond/10)
					log.Printf("job %s (type %s) completed after %s", idStr, name, roundDuration)
				}(name, start, idStr, queryCount)
				return nil
			} else if err != nil {
				continue
			}
			if qj.Attempts < currentAttemptCount {
				// Another thread decremented the attempt count and re-queued
				// the job, we're done.
				go metrics.Time(fmt.Sprintf("wait_for_job.%s.latency", name), time.Since(start))
				go metrics.Increment(fmt.Sprintf("wait_for_job.%s.attempt_count_decremented", name))
				log.Printf("job %s (type %s) failed after %v, retrying", idStr, name, time.Since(start))
				return nil
			}
		}
	}
}
