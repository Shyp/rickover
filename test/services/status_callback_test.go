package services

import (
	"testing"

	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/archived_jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/services"
	"github.com/Shyp/rickover/test"
	"github.com/Shyp/rickover/test/factory"
)

func TestStatusCallbackInsertsArchivedRecordDeletesQueuedRecord(t *testing.T) {
	defer test.TearDown(t)
	qj := factory.CreateQueuedJob(t, factory.EmptyData)
	err := services.HandleStatusCallback(qj.ID, "echo", models.StatusSucceeded, 7, true)
	test.AssertNotError(t, err, "")
	_, err = queued_jobs.Get(qj.ID)
	test.AssertEquals(t, err, queued_jobs.ErrNotFound)
	aj, err := archived_jobs.Get(qj.ID)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, aj.ID.String(), qj.ID.String())
	test.AssertEquals(t, aj.Attempts, uint8(7))
	test.AssertEquals(t, aj.Name, "echo")
	test.AssertEquals(t, aj.Status, models.StatusSucceeded)
}

func testStatusCallbackFailedInsertsArchivedRecord(t *testing.T) {
	t.Parallel()
	job, qj := factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	err := services.HandleStatusCallback(qj.ID, job.Name, models.StatusFailed, 1, true)
	test.AssertNotError(t, err, "")
	_, err = queued_jobs.Get(qj.ID)
	test.AssertEquals(t, err, queued_jobs.ErrNotFound)
	aj, err := archived_jobs.Get(qj.ID)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, aj.ID.String(), qj.ID.String())
}

func TestStatusCallbackFailedAtMostOnceInsertsArchivedRecord(t *testing.T) {
	defer test.TearDown(t)
	_, qj := factory.CreateAtMostOnceJob(t, factory.EmptyData)
	err := services.HandleStatusCallback(qj.ID, "at-most-once", models.StatusFailed, 7, true)
	test.AssertNotError(t, err, "")
	_, err = queued_jobs.Get(qj.ID)
	test.AssertEquals(t, err, queued_jobs.ErrNotFound)
	aj, err := archived_jobs.Get(qj.ID)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, aj.ID.String(), qj.ID.String())
}

func testStatusCallbackFailedAtLeastOnceUpdatesQueuedRecord(t *testing.T) {
	t.Parallel()
	job, qj := factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	err := services.HandleStatusCallback(qj.ID, job.Name, models.StatusFailed, 7, true)
	test.AssertNotError(t, err, "")

	qj, err = queued_jobs.Get(qj.ID)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, qj.Attempts, uint8(6))

	_, err = archived_jobs.Get(qj.ID)
	test.AssertEquals(t, err, archived_jobs.ErrNotFound)
}

func testStatusCallbackFailedNotRetryableArchivesRecord(t *testing.T) {
	t.Parallel()
	qj := factory.CreateQJ(t)
	err := services.HandleStatusCallback(qj.ID, qj.Name, models.StatusFailed, qj.Attempts, false)
	test.AssertNotError(t, err, "inserting archived record")

	_, err = queued_jobs.Get(qj.ID)
	test.AssertEquals(t, err, queued_jobs.ErrNotFound)
	aj, err := archived_jobs.Get(qj.ID)
	test.AssertNotError(t, err, "finding archived job")
	test.AssertEquals(t, aj.Status, models.StatusFailed)
	test.AssertEquals(t, aj.Attempts, qj.Attempts-1)
}

// This test returns an error - if the queued job doesn't exist, we can't
// create an archived job.
func TestStatusCallbackFailedAtMostOnceArchivedRecordExists(t *testing.T) {
	defer test.TearDown(t)
	aj := factory.CreateArchivedJob(t, factory.EmptyData, models.StatusFailed)
	err := services.HandleStatusCallback(aj.ID, aj.Name, models.StatusFailed, 1, true)
	test.AssertEquals(t, err, queued_jobs.ErrNotFound)
}
