package servertest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-types"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/archived_jobs"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/server"
	"github.com/Shyp/rickover/test"
	"github.com/Shyp/rickover/test/factory"
)

var u = &server.UnsafeBypassAuthorizer{}

var testPassword = "XmTGoDTRyVd8HHiuzFtPzF8N&or7ETPaPVvWuR;d"

func init() {
	server.DefaultAuthorizer.AddUser("test", testPassword)
}

func TestGoodRequestReturns200(t *testing.T) {
	defer test.TearDown(t)
	factory.CreateQueuedJob(t, factory.EmptyData)
	w := httptest.NewRecorder()
	a := uint8(3)
	jsr := &server.JobStatusRequest{
		Status:  "succeeded",
		Attempt: &a,
	}
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(jsr)
	req, _ := http.NewRequest("POST", "/v1/jobs/echo/job_6740b44e-13b9-475d-af06-979627e0e0d6", b)
	req.SetBasicAuth("foo", "bar")
	server.Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusOK)
}

func TestFailedUnretryableArchivesJob(t *testing.T) {
	t.Parallel()
	qj := factory.CreateQJ(t)
	w := httptest.NewRecorder()
	jsr := &server.JobStatusRequest{
		Status:    "failed",
		Retryable: func() *bool { b := false; return &b }(),
		Attempt:   &qj.Attempts,
	}
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(jsr)
	path := fmt.Sprintf("/v1/jobs/%s/%s", qj.Name, qj.ID.String())
	req, _ := http.NewRequest("POST", path, b)
	req.SetBasicAuth("foo", "bar")
	server.Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, 200)

	_, err := queued_jobs.Get(qj.ID)
	test.AssertEquals(t, err, queued_jobs.ErrNotFound)
	aj, err := archived_jobs.Get(qj.ID)
	test.AssertNotError(t, err, "finding archived job")
	test.AssertEquals(t, aj.Status, models.StatusFailed)
	test.AssertEquals(t, aj.Attempts, qj.Attempts-1)
}

var validRequest = server.CreateJobRequest{
	Name:             "email-signup",
	DeliveryStrategy: models.StrategyAtLeastOnce,
	Attempts:         7,
	Concurrency:      3,
}

func TestCreateJobReturnsJob(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(validRequest)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	test.AssertNotError(t, err, "")
	req.SetBasicAuth("foo", "bar")
	server.Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusCreated)
	job := new(models.Job)
	err = json.NewDecoder(w.Body).Decode(job)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, job.Name, validRequest.Name)
	test.AssertEquals(t, job.Attempts, validRequest.Attempts)
	test.AssertEquals(t, job.Concurrency, validRequest.Concurrency)
	test.AssertEquals(t, job.DeliveryStrategy, validRequest.DeliveryStrategy)
	diff := time.Since(job.CreatedAt)
	test.Assert(t, diff < 25*time.Millisecond, fmt.Sprintf("diff: %v created: %v", diff, job.CreatedAt))
}

func TestSuccessWritesDBRecord(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(validRequest)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	test.AssertNotError(t, err, "")
	req.SetBasicAuth("foo", "bar")
	server.Get(u).ServeHTTP(w, req)
	job, err := jobs.Get(validRequest.Name)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, job.Name, validRequest.Name)
	test.AssertEquals(t, job.Attempts, validRequest.Attempts)
	test.AssertEquals(t, job.Concurrency, validRequest.Concurrency)
	test.AssertEquals(t, job.DeliveryStrategy, validRequest.DeliveryStrategy)
	diff := time.Since(job.CreatedAt)
	test.Assert(t, diff < 50*time.Millisecond, fmt.Sprintf("insert took too long: %v\n", diff))
	name, offset := job.CreatedAt.Zone()
	test.AssertEquals(t, name, "UTC")
	test.AssertEquals(t, offset, 0)
}

func TestRetrieveJob(t *testing.T) {
	defer test.TearDown(t)
	factory.CreateQueuedJob(t, factory.EmptyData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/jobs/echo/job_6740b44e-13b9-475d-af06-979627e0e0d6", nil)
	req.SetBasicAuth("foo", "bar")
	server.Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusOK)
	var qj models.QueuedJob
	err := json.NewDecoder(w.Body).Decode(&qj)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, qj.ID.String(), "job_6740b44e-13b9-475d-af06-979627e0e0d6")
	test.AssertEquals(t, qj.Name, "echo")
	test.AssertEquals(t, qj.Status, models.StatusQueued)
}

func TestRetrieveJobNoName(t *testing.T) {
	defer test.TearDown(t)
	factory.CreateQueuedJob(t, factory.EmptyData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/jobs/job_6740b44e-13b9-475d-af06-979627e0e0d6", nil)
	req.SetBasicAuth("test", testPassword)
	server.DefaultServer.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusOK)
	var qj models.QueuedJob
	err := json.NewDecoder(w.Body).Decode(&qj)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, qj.ID.String(), "job_6740b44e-13b9-475d-af06-979627e0e0d6")
	test.AssertEquals(t, qj.Name, "echo")
	test.AssertEquals(t, qj.Status, models.StatusQueued)
}

func TestRetrieveArchivedJob(t *testing.T) {
	defer test.TearDown(t)
	factory.CreateArchivedJob(t, factory.EmptyData, models.StatusSucceeded)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/jobs/echo/job_6740b44e-13b9-475d-af06-979627e0e0d6", nil)
	req.SetBasicAuth("foo", "bar")
	server.Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusOK)
	var aj models.ArchivedJob
	err := json.NewDecoder(w.Body).Decode(&aj)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, aj.ID.String(), "job_6740b44e-13b9-475d-af06-979627e0e0d6")
	test.AssertEquals(t, aj.Name, "echo")
	test.AssertEquals(t, aj.Status, models.StatusSucceeded)
}

func TestReplayJob(t *testing.T) {
	defer test.TearDown(t)
	factory.CreateArchivedJob(t, factory.EmptyData, models.StatusSucceeded)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/jobs/echo/job_6740b44e-13b9-475d-af06-979627e0e0d6/replay", nil)
	req.SetBasicAuth("test", testPassword)
	server.DefaultServer.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, 201)
	var qj models.QueuedJob
	err := json.NewDecoder(w.Body).Decode(&qj)
	test.AssertNotError(t, err, "")
	test.AssertNotEquals(t, qj.ID.String(), "job_6740b44e-13b9-475d-af06-979627e0e0d6")
}

func TestReplayJobWithNoName(t *testing.T) {
	defer test.TearDown(t)
	factory.CreateArchivedJob(t, factory.EmptyData, models.StatusSucceeded)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/jobs/job_6740b44e-13b9-475d-af06-979627e0e0d6/replay", nil)
	req.SetBasicAuth("test", testPassword)
	server.DefaultServer.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, 201)
	var qj models.QueuedJob
	err := json.NewDecoder(w.Body).Decode(&qj)
	test.AssertNotError(t, err, "")
	test.AssertNotEquals(t, qj.ID.String(), "job_6740b44e-13b9-475d-af06-979627e0e0d6")
}

func TestReplayQueuedJobFails(t *testing.T) {
	defer test.TearDown(t)
	qj := factory.CreateQueuedJob(t, factory.EmptyData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/jobs/echo/%s/replay", qj.ID.String()), nil)
	req.SetBasicAuth("test", testPassword)
	server.DefaultServer.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, 400)
}

func Test202SuccessfulEnqueue(t *testing.T) {
	defer test.TearDown(t)
	_ = factory.CreateJob(t, factory.SampleJob)

	expiry := time.Now().UTC().Add(5 * time.Minute)
	w := httptest.NewRecorder()
	ejr := &server.EnqueueJobRequest{
		Data:      factory.EmptyData,
		ExpiresAt: types.NullTime{Valid: true, Time: expiry},
	}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(ejr)
	req, _ := http.NewRequest("PUT", "/v1/jobs/echo/job_6740b44e-13b9-475d-af06-979627e0e0d6", b)
	req.SetBasicAuth("test", testPassword)
	server.DefaultServer.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusAccepted)
	var j models.QueuedJob
	err := json.NewDecoder(w.Body).Decode(&j)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, j.ID.String(), "job_6740b44e-13b9-475d-af06-979627e0e0d6")
	test.AssertEquals(t, j.Attempts, uint8(7))
	test.AssertEquals(t, j.Status, models.StatusQueued)
	test.AssertEquals(t, j.Name, "echo")

	diff := j.ExpiresAt.Time.Sub(expiry)
	test.Assert(t, diff < 20*time.Millisecond, "")
	test.Assert(t, diff > -20*time.Millisecond, "")

	diff = time.Since(j.RunAfter)
	test.Assert(t, diff < 20*time.Millisecond, "")

	diff = time.Since(j.CreatedAt)
	test.Assert(t, diff < 20*time.Millisecond, "")

	diff = time.Since(j.UpdatedAt)
	test.Assert(t, diff < 20*time.Millisecond, "")
}

func Test202RandomId(t *testing.T) {
	defer test.TearDown(t)
	_ = factory.CreateJob(t, factory.SampleJob)

	w := httptest.NewRecorder()
	ejr := &server.EnqueueJobRequest{
		Data: factory.EmptyData,
	}
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(ejr)
	req, _ := http.NewRequest("PUT", "/v1/jobs/echo/random_id", b)
	req.SetBasicAuth("test", testPassword)
	server.DefaultServer.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusAccepted)
}

func Test202DuplicateEnqueue(t *testing.T) {
	defer test.TearDown(t)
	_ = factory.CreateJob(t, factory.SampleJob)

	w := httptest.NewRecorder()
	w2 := httptest.NewRecorder()
	ejr := &server.EnqueueJobRequest{
		Data: factory.EmptyData,
	}
	bits, _ := json.Marshal(ejr)
	req, _ := http.NewRequest("PUT", "/v1/jobs/echo/job_6740b44e-13b9-475d-af06-979627e0e0d6", bytes.NewReader(bits))
	req.SetBasicAuth("test", testPassword)
	server.DefaultServer.ServeHTTP(w, req)

	req, _ = http.NewRequest("PUT", "/v1/jobs/echo/job_6740b44e-13b9-475d-af06-979627e0e0d6", bytes.NewReader(bits))
	req.SetBasicAuth("test", testPassword)
	server.DefaultServer.ServeHTTP(w2, req)
	test.AssertEquals(t, w2.Code, http.StatusAccepted)
	var j models.QueuedJob
	err := json.NewDecoder(w.Body).Decode(&j)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, j.ID.String(), "job_6740b44e-13b9-475d-af06-979627e0e0d6")
}

func Test404JobNotFound(t *testing.T) {
	test.SetUp(t)
	t.Parallel()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/jobs/unknown", nil)
	req.SetBasicAuth("usr_123", "tok_123")
	server.Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusNotFound)
}

var sampleJob = models.Job{
	Attempts:         1,
	DeliveryStrategy: models.StrategyAtMostOnce,
	Concurrency:      1,
	Name:             "echo",
}

func Test200JobFound(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	_, err := jobs.Create(sampleJob)
	test.AssertNotError(t, err, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/jobs/echo", nil)
	req.SetBasicAuth("usr_123", "tok_123")
	server.Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusOK)
}

var validAtMostOnceRequest = server.CreateJobRequest{
	Name:             "email-signup",
	DeliveryStrategy: models.StrategyAtMostOnce,
	Attempts:         1,
	Concurrency:      3,
}

func TestCreateJobAtMostOnceSuccess(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	w := httptest.NewRecorder()
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(validAtMostOnceRequest)
	req, err := http.NewRequest("POST", "/v1/jobs", b)
	test.AssertNotError(t, err, "")
	req.SetBasicAuth("usr_123", "tok_123")
	server.Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusCreated)
}
