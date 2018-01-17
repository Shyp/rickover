package test_queued_jobs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Shyp/go-dberror"
	"github.com/Shyp/go-types"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/services"
	"github.com/Shyp/rickover/test"
	"github.com/Shyp/rickover/test/factory"
)

var empty = json.RawMessage([]byte("{}"))

var sampleJob = models.Job{
	Name:             "echo",
	DeliveryStrategy: models.StrategyAtLeastOnce,
	Attempts:         3,
	Concurrency:      1,
}

func TestAll(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	t.Run("Parallel", func(t *testing.T) {
		// Parallel tests go here
		t.Run("TestEnqueueUnknownJobTypeErrNoRows", testEnqueueUnknownJobTypeErrNoRows)
		t.Run("TestNonexistentReturnsErrNoRows", testNonexistentReturnsErrNoRows)
		t.Run("TestDeleteNonexistentJobReturnsErrNoRows", testDeleteNonexistentJobReturnsErrNoRows)
		t.Run("TestGetQueuedJob", testGetQueuedJob)
		t.Run("TestDeleteQueuedJob", testDeleteQueuedJob)
		t.Run("TestAcquireReturnsCorrectValues", testAcquireReturnsCorrectValues)
		t.Run("TestEnqueueNoData", testEnqueueNoData)
		t.Run("EnqueueWithExistingArchivedJobFails", testEnqueueWithExistingArchivedJobFails)
	})
}

func TestEnqueue(t *testing.T) {
	defer test.TearDown(t)
	qj := factory.CreateQueuedJob(t, factory.EmptyData)
	test.AssertEquals(t, qj.ID.String(), "job_6740b44e-13b9-475d-af06-979627e0e0d6")
	test.AssertEquals(t, qj.Name, "echo")
	test.AssertEquals(t, qj.Attempts, uint8(7))
	test.AssertEquals(t, qj.Status, models.StatusQueued)

	diff := time.Since(qj.RunAfter)
	test.Assert(t, diff < 100*time.Millisecond, "")

	diff = time.Since(qj.CreatedAt)
	test.Assert(t, diff < 100*time.Millisecond, "")

	diff = time.Since(qj.UpdatedAt)
	test.Assert(t, diff < 100*time.Millisecond, "")
}

func testEnqueueNoData(t *testing.T) {
	t.Parallel()
	id, _ := types.GenerateUUID("jobname_")
	j := models.Job{
		Name:             id.String(),
		DeliveryStrategy: models.StrategyAtLeastOnce,
		Attempts:         7,
		Concurrency:      1,
	}
	_, err := jobs.Create(j)
	test.AssertNotError(t, err, "")

	expiresAt := types.NullTime{Valid: false}
	runAfter := time.Now().UTC()

	qjid, _ := types.GenerateUUID("job_")
	_, err = queued_jobs.Enqueue(qjid, j.Name, runAfter, expiresAt, []byte{})
	test.AssertError(t, err, "")
	switch terr := err.(type) {
	case *dberror.Error:
		test.AssertEquals(t, terr.Message, "Invalid input syntax for type json")
	default:
		t.Fatalf("Expected a dberror, got %#v", terr)
	}
}

func TestEnqueueJobExists(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	_, err := jobs.Create(sampleJob)
	test.AssertNotError(t, err, "")

	expiresAt := types.NullTime{Valid: false}
	runAfter := time.Now().UTC()

	_, err = queued_jobs.Enqueue(factory.JobId, "echo", runAfter, expiresAt, empty)
	test.AssertNotError(t, err, "")
	_, err = queued_jobs.Enqueue(factory.JobId, "echo", runAfter, expiresAt, empty)
	test.AssertError(t, err, "")
	switch terr := err.(type) {
	case *dberror.Error:
		test.AssertEquals(t, terr.Code, dberror.CodeUniqueViolation)
		test.AssertEquals(t, terr.Column, "id")
		test.AssertEquals(t, terr.Table, "queued_jobs")
		test.AssertEquals(t, terr.Message,
			fmt.Sprintf("A id already exists with this value (6740b44e-13b9-475d-af06-979627e0e0d6)"))
	default:
		t.Fatalf("Expected a dberror, got %#v", terr)
	}
}

func testEnqueueUnknownJobTypeErrNoRows(t *testing.T) {
	t.Parallel()

	expiresAt := types.NullTime{Valid: false}
	runAfter := time.Now().UTC()
	_, err := queued_jobs.Enqueue(factory.JobId, "unknownJob", runAfter, expiresAt, empty)
	test.AssertError(t, err, "")
	test.AssertEquals(t, err.Error(), "Job type unknownJob does not exist or the job with that id has already been archived")
}

func testEnqueueWithExistingArchivedJobFails(t *testing.T) {
	t.Parallel()
	_, qj := factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	err := services.HandleStatusCallback(qj.ID, qj.Name, models.StatusSucceeded, qj.Attempts, true)
	test.AssertNotError(t, err, "")
	expiresAt := types.NullTime{Valid: false}
	runAfter := time.Now().UTC()
	_, err = queued_jobs.Enqueue(qj.ID, qj.Name, runAfter, expiresAt, empty)
	test.AssertError(t, err, "")
	test.AssertEquals(t, err.Error(), "Job type "+qj.Name+" does not exist or the job with that id has already been archived")
}

func testNonexistentReturnsErrNoRows(t *testing.T) {
	t.Parallel()
	id, _ := types.NewPrefixUUID("job_a9173b65-7714-42b4-85f2-8336f6d12180")
	_, err := queued_jobs.Get(id)
	test.AssertEquals(t, err, queued_jobs.ErrNotFound)
}

func testGetQueuedJob(t *testing.T) {
	t.Parallel()
	_, qj := factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	gotQj, err := queued_jobs.Get(qj.ID)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, gotQj.ID.String(), qj.ID.String())
}

func testDeleteQueuedJob(t *testing.T) {
	t.Parallel()
	_, qj := factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	err := queued_jobs.Delete(qj.ID)
	test.AssertNotError(t, err, "")
}

func testDeleteNonexistentJobReturnsErrNoRows(t *testing.T) {
	t.Parallel()
	err := queued_jobs.Delete(factory.RandomId("job_"))
	test.AssertEquals(t, err, queued_jobs.ErrNotFound)
}

func TestDataRoundtrip(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	_, err := jobs.Create(sampleJob)
	test.AssertNotError(t, err, "")

	type RemoteAccounts struct {
		ID string
	}

	type User struct {
		ID             string
		Balance        uint64
		CreatedAt      time.Time
		RemoteAccounts RemoteAccounts
		Pickups        []string
	}
	user := &User{
		ID:        "usr_123",
		Balance:   uint64(365),
		CreatedAt: time.Now().UTC(),
		RemoteAccounts: RemoteAccounts{
			ID: "rem_123",
		},
		Pickups: []string{"pik_123", "pik_234"},
	}

	expiresAt := types.NullTime{Valid: false}
	runAfter := time.Now().UTC()
	var d json.RawMessage
	d, err = json.Marshal(user)
	test.AssertNotError(t, err, "")
	qj, err := queued_jobs.Enqueue(factory.JobId, "echo", runAfter, expiresAt, d)
	test.AssertNotError(t, err, "")

	gotQj, err := queued_jobs.Get(qj.ID)
	test.AssertNotError(t, err, "")

	var u User
	err = json.Unmarshal(gotQj.Data, &u)
	test.AssertNotError(t, err, "expected to be able to convert data to a User object")
	test.AssertEquals(t, u.ID, "usr_123")
	test.AssertEquals(t, u.Balance, uint64(365))
	test.AssertEquals(t, u.RemoteAccounts.ID, "rem_123")
	test.AssertEquals(t, u.Pickups[0], "pik_123")
	test.AssertEquals(t, u.Pickups[1], "pik_234")
	test.AssertEquals(t, len(u.Pickups), 2)

	diff := time.Since(u.CreatedAt)
	test.Assert(t, diff < 100*time.Millisecond, "")
}

func testAcquireReturnsCorrectValues(t *testing.T) {
	t.Parallel()
	job, qj := factory.CreateUniqueQueuedJob(t, factory.EmptyData)

	gotQj, err := queued_jobs.Acquire(job.Name)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, gotQj.ID.String(), qj.ID.String())
	test.AssertEquals(t, gotQj.Status, models.StatusInProgress)
}

func TestAcquireTwoThreads(t *testing.T) {
	var wg sync.WaitGroup
	defer test.TearDown(t)
	factory.CreateQueuedJob(t, factory.EmptyData)

	wg.Add(2)
	var err1, err2 error
	var gotQj1, gotQj2 *models.QueuedJob
	go func() {
		gotQj1, err1 = queued_jobs.Acquire(sampleJob.Name)
		wg.Done()
	}()
	go func() {
		gotQj2, err2 = queued_jobs.Acquire(sampleJob.Name)
		wg.Done()
	}()

	wg.Wait()
	test.Assert(t, err1 == sql.ErrNoRows || err2 == sql.ErrNoRows, "expected one error to be ErrNoRows")
	test.Assert(t, gotQj1 != nil || gotQj2 != nil, "expected one job to be acquired")
}

func TestAcquireDoesntGetFutureJob(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)

	_, err := jobs.Create(sampleJob)
	test.AssertNotError(t, err, "")

	expiresAt := types.NullTime{Valid: false}
	runAfter := time.Now().UTC().Add(20 * time.Millisecond)
	qj, err := queued_jobs.Enqueue(factory.JobId, "echo", runAfter, expiresAt, empty)
	test.AssertNotError(t, err, "")
	_, err = queued_jobs.Acquire(qj.Name)
	test.AssertEquals(t, err, sql.ErrNoRows)
}

func TestAcquireDoesntGetInProgressJob(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)

	_, err := jobs.Create(sampleJob)
	test.AssertNotError(t, err, "")

	expiresAt := types.NullTime{Valid: false}
	runAfter := time.Now().UTC()
	qj, err := queued_jobs.Enqueue(factory.JobId, "echo", runAfter, expiresAt, empty)
	test.AssertNotError(t, err, "")
	qj, err = queued_jobs.Acquire(qj.Name)
	test.AssertNotError(t, err, "")
	test.AssertDeepEquals(t, qj.ID, factory.JobId)

	_, err = queued_jobs.Acquire(qj.Name)
	test.AssertEquals(t, err, sql.ErrNoRows)
}

func TestDecrementDecrements(t *testing.T) {
	defer test.TearDown(t)
	qj := factory.CreateQueuedJob(t, factory.EmptyData)
	qj, err := queued_jobs.Decrement(qj.ID, 7, time.Now().Add(1*time.Minute))
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, qj.Attempts, uint8(6))
	test.AssertBetween(t, int64(qj.RunAfter.Sub(time.Now())), int64(59*time.Second), int64(1*time.Minute))
}

func TestDecrementErrNoRowsWrongAttempts(t *testing.T) {
	defer test.TearDown(t)
	qj := factory.CreateQueuedJob(t, factory.EmptyData)
	_, err := queued_jobs.Decrement(qj.ID, 1, time.Now())
	test.AssertEquals(t, err, sql.ErrNoRows)
}

func TestCountAll(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	allCount, readyCount, err := queued_jobs.CountReadyAndAll()
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, allCount, 0)
	test.AssertEquals(t, readyCount, 0)

	factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	allCount, readyCount, err = queued_jobs.CountReadyAndAll()
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, allCount, 3)
	test.AssertEquals(t, readyCount, 3)
}

func TestCountByStatus(t *testing.T) {
	defer test.TearDown(t)
	job, _ := factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	factory.CreateQueuedJobOnly(t, job.Name, factory.EmptyData)
	factory.CreateQueuedJobOnly(t, job.Name, factory.EmptyData)
	factory.CreateAtMostOnceJob(t, factory.EmptyData)
	m, err := queued_jobs.GetCountsByStatus(models.StatusQueued)
	test.AssertNotError(t, err, "")
	test.Assert(t, len(m) >= 2, "expected at least 2 queued jobs in the database")
	test.AssertEquals(t, m[job.Name], int64(3))
	test.AssertEquals(t, m["at-most-once"], int64(1))
}

func TestOldInProgress(t *testing.T) {
	defer test.TearDown(t)
	_, qj1 := factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	_, qj2 := factory.CreateUniqueQueuedJob(t, factory.EmptyData)
	_, err := queued_jobs.Acquire(qj1.Name)
	test.AssertNotError(t, err, "")
	_, err = queued_jobs.Acquire(qj2.Name)
	test.AssertNotError(t, err, "")
	jobs, err := queued_jobs.GetOldInProgressJobs(time.Now().UTC().Add(40 * time.Millisecond))
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, len(jobs), 2)
	if jobs[0].ID.String() == qj1.ID.String() {
		test.AssertEquals(t, jobs[1].ID.String(), qj2.ID.String())
	} else {
		test.AssertEquals(t, jobs[1].ID.String(), qj1.ID.String())
	}
	jobs, err = queued_jobs.GetOldInProgressJobs(time.Now().UTC().Add(-1 * time.Second))
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, len(jobs), 0)
}
