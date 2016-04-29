// Factories for building objects for tests.
package factory

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-dberror"
	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-types"
	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/nu7hatch/gouuid"
	"github.com/Shyp/rickover/downstream"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/archived_jobs"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/services"
	"github.com/Shyp/rickover/test"
	"github.com/Shyp/rickover/test/db"
)

var EmptyData = json.RawMessage([]byte("{}"))

var JobId types.PrefixUUID

func init() {
	id, _ := types.NewPrefixUUID("job_6740b44e-13b9-475d-af06-979627e0e0d6")
	JobId = id
}

type RandomData struct {
	Foo []string `json:"foo"`
	Baz uint8    `json:"baz"`
}

var RD = &RandomData{
	Foo: []string{"bar", "pik_345"},
	Baz: uint8(17),
}

var SampleJob = models.Job{
	Name:             "echo",
	DeliveryStrategy: models.StrategyAtLeastOnce,
	Attempts:         7,
	Concurrency:      1,
}

var SampleAtMostOnceJob = models.Job{
	Name:             "at-most-once",
	DeliveryStrategy: models.StrategyAtMostOnce,
	Attempts:         1,
	Concurrency:      5,
}

// RandomId returns a random UUID with the given prefix.
func RandomId(prefix string) types.PrefixUUID {
	id, _ := uuid.NewV4()
	return types.PrefixUUID{
		UUID:   id,
		Prefix: prefix,
	}
}

func CreateJob(t *testing.T, j models.Job) models.Job {
	db.SetUp(t)
	job, err := jobs.Create(j)
	test.AssertNotError(t, err, "")
	return *job
}

// CreateQueuedJob creates a job and a queued job with the given JSON data, and
// returns the created queued job.
func CreateQueuedJob(t *testing.T, data json.RawMessage) *models.QueuedJob {
	return createJobAndQueuedJob(t, SampleJob, data, false)
}

// CreateRandomQueuedJob creates a queued job with a random UUID.
func CreateRandomQueuedJob(t *testing.T, data json.RawMessage) *models.QueuedJob {
	return createJobAndQueuedJob(t, SampleJob, data, true)
}

func CreateArchivedJob(t *testing.T, data json.RawMessage, status models.JobStatus) *models.ArchivedJob {
	qj := createJobAndQueuedJob(t, SampleJob, data, false)
	aj, err := archived_jobs.Create(qj.Id, qj.Name, models.StatusSucceeded, qj.Attempts)
	test.AssertNotError(t, err, "")
	err = queued_jobs.DeleteRetry(qj.Id, 3)
	test.AssertNotError(t, err, "")
	return aj
}

// CreateAtMostOnceJob creates a queued job that can be run at most once.
func CreateAtMostOnceJob(t *testing.T, data json.RawMessage) *models.QueuedJob {
	return createJobAndQueuedJob(t, SampleAtMostOnceJob, data, false)
}

func createJobAndQueuedJob(t *testing.T, j models.Job, data json.RawMessage, randomId bool) *models.QueuedJob {
	db.SetUp(t)
	_, err := jobs.Create(j)
	if err != nil {
		switch dberr := err.(type) {
		case *dberror.Error:
			if dberr.Code == dberror.CodeUniqueViolation {
			} else {
				test.AssertNotError(t, err, "")
			}
		default:
			test.AssertNotError(t, err, "")
		}
	}

	expiresAt := types.NullTime{Valid: false}
	runAfter := time.Now().UTC()
	var id types.PrefixUUID
	if randomId {
		id = RandomId("job_")
	} else {
		id = JobId
	}
	qj, err := queued_jobs.Enqueue(id, j.Name, runAfter, expiresAt, data)
	test.AssertNotError(t, err, "")
	return qj
}

// Processor returns a simple JobProcessor, with a client pointing at the given
// URL.
func Processor(url string) *services.JobProcessor {
	jp := &services.JobProcessor{
		Client: downstream.NewClient("jobs", "password", url),
	}
	return jp
}
