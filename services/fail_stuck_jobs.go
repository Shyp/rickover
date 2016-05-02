package services

import (
	"log"
	"time"

	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/queued_jobs"
)

// ArchiveStuckJobs marks as failed any queued jobs with an updated_at
// timestamp older than the olderThan value.
func ArchiveStuckJobs(olderThan time.Duration) error {
	var olderThanTime time.Time
	if olderThan >= 0 {
		olderThanTime = time.Now().Add(-1 * olderThan)
	} else {
		olderThanTime = time.Now().Add(olderThan)
	}
	jobs, err := queued_jobs.GetOldInProgressJobs(olderThanTime)
	if err != nil {
		return err
	}
	for _, qj := range jobs {
		err = HandleStatusCallback(qj.Id, qj.Name, models.StatusFailed, qj.Attempts)
		if err == nil {
			log.Printf("Found stuck job %s and marked it as failed", qj.Id.String())
		} else {
			// We don't want to return an error here since there may easily be
			// race/idempotence errors with a stuck job watcher. If it errors
			// we'll grab it with the next cron.
			log.Printf("Found stuck job %s but could not process it: %s", qj.Id.String(), err.Error())
		}
	}
	return nil
}

// WatchStuckJobs polls the queued_jobs table for stuck jobs (defined as
// in-progress jobs that haven't been updated in oldDuration time), and marks
// them as failed.
func WatchStuckJobs(interval time.Duration, olderThan time.Duration) {
	for _ = range time.Tick(interval) {
		go func() {
			err := ArchiveStuckJobs(olderThan)
			if err != nil {
				log.Printf("Error archiving stuck jobs: %s\n", err.Error())
			}
		}()
	}
}
