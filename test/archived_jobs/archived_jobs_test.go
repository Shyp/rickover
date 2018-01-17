package test_archived_jobs

import (
	"testing"
	"time"

	"github.com/Shyp/go-dberror"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/archived_jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/test"
	"github.com/Shyp/rickover/test/factory"
)

var sampleJob = models.Job{
	Name:             "echo",
	DeliveryStrategy: models.StrategyAtLeastOnce,
	Attempts:         3,
	Concurrency:      1,
}

// Test that creating an archived job returns the job
func TestCreateJobReturnsJob(t *testing.T) {
	t.Parallel()
	qj := factory.CreateQJ(t)
	aj, err := archived_jobs.Create(qj.ID, qj.Name, models.StatusSucceeded, qj.Attempts)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, aj.ID.String(), qj.ID.String())
	test.AssertEquals(t, aj.Status, models.StatusSucceeded)
	test.AssertEquals(t, aj.Attempts, uint8(qj.Attempts))
	test.AssertEquals(t, string(aj.Data), "{\"baz\": 17, \"foo\": [\"bar\", \"pik_345\"]}")
	test.AssertEquals(t, aj.ExpiresAt.Valid, true)
	test.AssertEquals(t, aj.ExpiresAt.Time, qj.ExpiresAt.Time)

	diff := time.Since(aj.CreatedAt)
	test.Assert(t, diff < 20*time.Millisecond, "")
}

// Test that creating an archived job when one already exists returns
// a uniqueness constraint failure.
func TestArchivedJobFailsIfJobExists(t *testing.T) {
	qj := factory.CreateQueuedJob(t, factory.EmptyData)
	defer test.TearDown(t)
	_, err := archived_jobs.Create(qj.ID, "echo", models.StatusSucceeded, 7)
	test.AssertNotError(t, err, "")
	_, err = archived_jobs.Create(qj.ID, "echo", models.StatusSucceeded, 7)
	test.AssertError(t, err, "")
	switch terr := err.(type) {
	case *dberror.Error:
		test.AssertEquals(t, terr.Code, dberror.CodeUniqueViolation)
		test.AssertEquals(t, terr.Column, "id")
		test.AssertEquals(t, terr.Table, "archived_jobs")
		test.AssertEquals(t, terr.Message,
			"A id already exists with this value (6740b44e-13b9-475d-af06-979627e0e0d6)")
	default:
		t.Fatalf("Expected a dberror, got %#v", terr)
	}
}

// Test that creating a job stores the data in the database
func TestCreateJobStoresJob(t *testing.T) {
	qj := factory.CreateQueuedJob(t, factory.EmptyData)
	defer test.TearDown(t)
	aj, err := archived_jobs.Create(qj.ID, "echo", models.StatusSucceeded, 7)
	test.AssertNotError(t, err, "")
	aj, err = archived_jobs.Get(aj.ID)
	test.AssertNotError(t, err, "")

	test.AssertEquals(t, aj.ID.String(), qj.ID.String())
	test.AssertEquals(t, aj.Status, models.StatusSucceeded)
	test.AssertEquals(t, aj.Attempts, uint8(7))
	test.AssertEquals(t, string(aj.Data), "{}")

	diff := time.Since(aj.CreatedAt)
	test.Assert(t, diff < 20*time.Millisecond, "")
}

// Test that creating an archived job when the job does not exist in QueuedJobs
// returns sql.ErrNoRows
func TestCreateArchivedJobWithNoQueuedReturnsErrNoRows(t *testing.T) {
	t.Parallel()
	test.SetUp(t)
	_, err := archived_jobs.Create(factory.JobId, "echo", models.StatusSucceeded, 7)
	test.AssertEquals(t, err, queued_jobs.ErrNotFound)
}

// Test that creating an archived job when the job does not exist in QueuedJobs
// returns sql.ErrNoRows
func TestCreateArchivedJobWithWrongNameReturnsErrNoRows(t *testing.T) {
	qj := factory.CreateQueuedJob(t, factory.EmptyData)
	defer test.TearDown(t)
	_, err := archived_jobs.Create(qj.ID, "wrong-job-name", models.StatusSucceeded, 7)
	test.AssertEquals(t, err, queued_jobs.ErrNotFound)
}
