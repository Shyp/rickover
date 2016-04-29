package services

import (
	"log"
	"time"

	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/queued_jobs"
)

func ArchiveStuckJobs(oldDefinition time.Duration) error {
	var olderThan time.Time
	if oldDefinition >= 0 {
		olderThan = time.Now().Add(-1 * oldDefinition)
	} else {
		olderThan = time.Now().Add(oldDefinition)
	}
	jobs, err := queued_jobs.GetOldInProgressJobs(olderThan)
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
func WatchStuckJobs(interval time.Duration, oldDefinition time.Duration) {
	for _ = range time.Tick(interval) {
		err := ArchiveStuckJobs(oldDefinition)
		if err != nil {
			log.Printf("Error archiving stuck jobs: %s\n", err.Error())
		}
	}
}
