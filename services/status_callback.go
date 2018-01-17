// Mediation layer between the server and database queries.
//
// Logic that's not related to validating request input/turning errors into
// HTTP responses should go here.
package services

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/Shyp/go-dberror"
	"github.com/Shyp/go-simple-metrics"
	"github.com/Shyp/go-types"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/archived_jobs"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
)

// HandleStatusCallback updates a queued job with the provided status and
// the attempts remaining. Likely the job will either be inserted into
// archived_jobs and removed from queued_jobs, or the job will have its
// attempts counter decremented in queued_jobs.
//
// This can return an error if any of the following happens: the archived_job
// already exists, the queued job no longer exists by the time you attempt to
// delete it, the number of attempts for the queued job don't match up with the
// passed in value (slow)
func HandleStatusCallback(id types.PrefixUUID, name string, status models.JobStatus, attempt uint8, retryable bool) error {
	if status == models.StatusSucceeded {
		err := createAndDelete(id, name, models.StatusSucceeded, attempt)
		if err != nil {
			go metrics.Increment("archived_job.create.success.error")
		} else {
			go metrics.Increment(fmt.Sprintf("archived_job.create.%s.success", name))
			go metrics.Increment("archived_job.create.success")
			go metrics.Increment("archived_job.create")
		}
		return err
	} else if status == models.StatusFailed {
		err := handleFailedCallback(id, name, attempt, retryable)
		if err != nil {
			go metrics.Increment("archived_job.create.failed.error")
		} else {
			go metrics.Increment(fmt.Sprintf("archived_job.create.%s.failed", name))
			go metrics.Increment("archived_job.create.failed")
			go metrics.Increment("archived_job.create")
		}
		return err
	} else {
		return fmt.Errorf("Unknown job status: %s", status)
	}
}

// createAndDelete creates an archived job, deletes the queued job, and returns
// any errors.
func createAndDelete(id types.PrefixUUID, name string, status models.JobStatus, attempt uint8) error {
	start := time.Now()
	_, err := archived_jobs.Create(id, name, status, attempt)
	go metrics.Time("archived_job.create.latency", time.Since(start))
	if err != nil {
		switch derr := err.(type) {
		case *dberror.Error:
			if derr.Code == dberror.CodeUniqueViolation {
				// Some other thread beat us to it. Don't return an error, just
				// fall through and try to delete the record.
				log.Printf("Could not create archived job %s with status %s because "+
					"it was already present. Deleting the queued job.", id.String(), status)
			} else {
				return err
			}
		default:
			return err
		}
	}
	start = time.Now()
	err = queued_jobs.DeleteRetry(id, 3)
	go metrics.Time("queued_job.delete.latency", time.Since(start))
	return err
}

// getRunAfter gets the time this job should run after, given the current
// attempt number and the attempts remaining.
func getRunAfter(totalAttempts, remainingAttempts uint8) time.Time {
	backoff := totalAttempts - remainingAttempts
	return time.Now().UTC().Add(time.Duration(math.Pow(2, float64(backoff))) * time.Second)
}

func handleFailedCallback(id types.PrefixUUID, name string, attempt uint8, retryable bool) error {
	remainingAttempts := attempt - 1
	if retryable == false || remainingAttempts == 0 {
		return createAndDelete(id, name, models.StatusFailed, remainingAttempts)
	}
	job, err := jobs.GetRetry(name, 3)
	if err != nil {
		return err
	}
	if job.DeliveryStrategy == models.StrategyAtMostOnce {
		return createAndDelete(id, name, models.StatusFailed, remainingAttempts)
	} else {
		// Try the job again. Note the database decrements the attempt counter
		start := time.Now()
		runAfter := getRunAfter(job.Attempts, remainingAttempts)
		_, err := queued_jobs.Decrement(id, attempt, runAfter)
		go metrics.Time("queued_jobs.decrement.latency", time.Since(start))
		return err
	}
}
