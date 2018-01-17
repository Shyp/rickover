// Package factory contains helpers for instantiating tests.
package factory

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Shyp/go-dberror"
	"github.com/Shyp/go-types"
	"github.com/Shyp/rickover/downstream"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/archived_jobs"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/services"
	"github.com/Shyp/rickover/test"
	"github.com/nu7hatch/gouuid"
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
	id, err := uuid.NewV4()
	if err != nil {
		panic(err.Error())
	}
	return types.PrefixUUID{
		UUID:   id,
		Prefix: prefix,
	}
}

func CreateJob(t testing.TB, j models.Job) models.Job {
	test.SetUp(t)
	job, err := jobs.Create(j)
	test.AssertNotError(t, err, "")
	return *job
}

// CreateQueuedJob creates a job and a queued job with the given JSON data, and
// returns the created queued job.
func CreateQueuedJob(t testing.TB, data json.RawMessage) *models.QueuedJob {
	t.Helper()
	_, qj := createJobAndQueuedJob(t, SampleJob, data, false)
	return qj
}

// Like the above but with unique ID's and job names
func CreateUniqueQueuedJob(t testing.TB, data json.RawMessage) (*models.Job, *models.QueuedJob) {
	id, _ := types.GenerateUUID("jobname_")
	j := models.Job{
		Name:             id.String(),
		DeliveryStrategy: models.StrategyAtLeastOnce,
		Attempts:         7,
		Concurrency:      1,
	}
	return createJobAndQueuedJob(t, j, data, true)
}

func CreateQueuedJobOnly(t testing.TB, name string, data json.RawMessage) *models.QueuedJob {
	t.Helper()
	expiresAt := types.NullTime{Valid: false}
	runAfter := time.Now().UTC()
	id := RandomId("job_")
	qj, err := queued_jobs.Enqueue(id, name, runAfter, expiresAt, data)
	test.AssertNotError(t, err, "")
	return qj
}

// CreateQJ creates a job with a random name, and a random UUID.
func CreateQJ(t testing.TB) *models.QueuedJob {
	t.Helper()
	test.SetUp(t)
	jobname := RandomId("jobtype")
	job, err := jobs.Create(models.Job{
		Name:             jobname.String(),
		Attempts:         11,
		Concurrency:      3,
		DeliveryStrategy: models.StrategyAtLeastOnce,
	})
	test.AssertNotError(t, err, "create job failed")
	now := time.Now().UTC()
	expires := types.NullTime{
		Time:  now.Add(5 * time.Minute),
		Valid: true,
	}
	dat, err := json.Marshal(RD)
	test.AssertNotError(t, err, "marshaling RD")
	qj, err := queued_jobs.Enqueue(RandomId("job_"), job.Name, now, expires, dat)
	test.AssertNotError(t, err, "create job failed")
	return qj
}

func CreateArchivedJob(t *testing.T, data json.RawMessage, status models.JobStatus) *models.ArchivedJob {
	t.Helper()
	_, qj := createJobAndQueuedJob(t, SampleJob, data, false)
	aj, err := archived_jobs.Create(qj.ID, qj.Name, models.StatusSucceeded, qj.Attempts)
	test.AssertNotError(t, err, "")
	err = queued_jobs.DeleteRetry(qj.ID, 3)
	test.AssertNotError(t, err, "")
	return aj
}

// CreateAtMostOnceJob creates a queued job that can be run at most once.
func CreateAtMostOnceJob(t *testing.T, data json.RawMessage) (*models.Job, *models.QueuedJob) {
	t.Helper()
	return createJobAndQueuedJob(t, SampleAtMostOnceJob, data, false)
}

func createJobAndQueuedJob(t testing.TB, j models.Job, data json.RawMessage, randomId bool) (*models.Job, *models.QueuedJob) {
	test.SetUp(t)
	job, err := jobs.Create(j)
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
	test.AssertNotError(t, err, fmt.Sprintf("Error creating queued job %s (job name %s)", id, j.Name))
	return job, qj
}

// Processor returns a simple JobProcessor, with a client pointing at the given
// URL, and various sleeps set to 0.
func Processor(url string) *services.JobProcessor {
	return &services.JobProcessor{
		Client:      downstream.NewClient("jobs", "password", url),
		Timeout:     200 * time.Millisecond,
		SleepFactor: 0,
	}
}
