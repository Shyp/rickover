package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Shyp/go-types"
	"github.com/Shyp/rest"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/archived_jobs"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/services"
	"github.com/Shyp/rickover/test"
	"github.com/Shyp/rickover/test/factory"
	"github.com/nu7hatch/gouuid"
)

func TestAll(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	t.Run("Parallel", func(t *testing.T) {
		t.Run("ExpiredJobNotEnqueued", testExpiredJobNotEnqueued)
		t.Run("StatusCallbackFailedNotRetryableArchivesRecord", testStatusCallbackFailedNotRetryableArchivesRecord)
		t.Run("StatusCallbackFailedAtLeastOnceUpdatesQueuedRecord", testStatusCallbackFailedAtLeastOnceUpdatesQueuedRecord)
		t.Run("TestStatusCallbackFailedInsertsArchivedRecord", testStatusCallbackFailedInsertsArchivedRecord)
	})
}

func testExpiredJobNotEnqueued(t *testing.T) {
	t.Parallel()

	c1 := make(chan bool, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		c1 <- true
	}))
	defer s.Close()
	jp := services.NewJobProcessor("password", s.URL)

	_, err := jobs.Create(factory.SampleJob)
	test.AssertNotError(t, err, "")
	expiresAt := types.NullTime{
		Valid: true,
		Time:  time.Now().UTC().Add(-5 * time.Millisecond),
	}
	qj, err := queued_jobs.Enqueue(factory.JobId, "echo", time.Now().UTC(), expiresAt, factory.EmptyData)
	test.AssertNotError(t, err, "")
	err = jp.DoWork(qj)
	test.AssertNotError(t, err, "")
	for {
		select {
		case <-c1:
			t.Fatalf("worker made a request to the server")
			return
		case <-time.After(60 * time.Millisecond):
			return
		}
	}
}

// 1. Create a job type
// 2. Enqueue a job
// 3. Create a test server that replies with a 503 rest.Error
// 4. Ensure that the worker retries
func TestWorkerRetriesJSON503(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)

	// make the test go faster
	originalSleepWorker503Factor := services.UnavailableSleepFactor
	services.UnavailableSleepFactor = 0
	defer func() {
		services.UnavailableSleepFactor = originalSleepWorker503Factor
	}()

	_, err := jobs.Create(factory.SampleJob)
	test.AssertNotError(t, err, "")

	id, _ := uuid.NewV4()
	pid, _ := types.NewPrefixUUID(fmt.Sprintf("job_%s", id))

	var data json.RawMessage
	data, err = json.Marshal(factory.RD)
	test.AssertNotError(t, err, "")
	qj, err := queued_jobs.Enqueue(pid, "echo", time.Now(), types.NullTime{Valid: false}, data)
	test.AssertNotError(t, err, "")

	var mu sync.Mutex
	count := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer func() {
			mu.Unlock()
		}()
		if count == 0 || count == 1 {
			count++
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(&rest.Error{
				Title:  "Service Unavailable",
				Detail: "The server will be shutting down momentarily and cannot accept new work.",
				ID:     "service_unavailable",
			})
		} else {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusAccepted)
			_, err = w.Write([]byte("{}"))
			test.AssertNotError(t, err, "")

			// Cheating, hit the internal success callback.
			callbackErr := services.HandleStatusCallback(qj.ID, "echo", models.StatusSucceeded, uint8(5), true)
			test.AssertNotError(t, callbackErr, "")
		}
	}))
	defer s.Close()
	jp := factory.Processor(s.URL)
	err = jp.DoWork(qj)
	test.AssertNotError(t, err, "")
}

func TestWorkerWaitsConnectTimeout(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	jp := services.NewJobProcessor("http://10.255.255.1", "password")

	// Okay this is not the world's best design.
	// Job processor client -> worker client -> generic rest client
	jp.Client.Client.Client.Timeout = 5 * time.Millisecond

	qj := factory.CreateQueuedJob(t, factory.EmptyData)
	go func() {
		err := services.HandleStatusCallback(qj.ID, qj.Name, models.StatusSucceeded, qj.Attempts, true)
		test.AssertNotError(t, err, "")
	}()

	// If this *doesn't* hit a timeout it'll hit HandleStatusCallback(failed),
	// which will throw an error.
	err := jp.DoWork(qj)
	test.AssertNotError(t, err, "")
}

// this could probably be a simpler test
func TestWorkerWaitsRequestTimeout(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	var wg sync.WaitGroup
	wg.Add(1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(60 * time.Millisecond)
		wg.Done()
	}))
	defer s.Close()

	jp := services.NewJobProcessor(s.URL, "password")

	// Okay this is not the world's best design.
	// Job processor client -> worker client -> generic rest client
	jp.Client.Client.Client.Timeout = 30 * time.Millisecond

	qj := factory.CreateQueuedJob(t, factory.EmptyData)
	go func() {
		err := services.HandleStatusCallback(qj.ID, qj.Name, models.StatusSucceeded, qj.Attempts, true)
		test.AssertNotError(t, err, "")
	}()

	workErr := jp.DoWork(qj)
	test.AssertNotError(t, workErr, "")
	wg.Wait()
	aj, err := archived_jobs.Get(qj.ID)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, aj.Status, models.StatusSucceeded)
}

func TestWorkerDoesNotWaitConnectionFailure(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	jp := services.NewJobProcessor(
		"password",
		// TODO empty port finder
		"http://127.0.0.1:29656",
	)

	// Okay this is not the world's best design.
	// Job processor client -> worker client -> generic rest client
	jp.Client.Client.Client.Timeout = 20 * time.Millisecond

	_, qj := factory.CreateAtMostOnceJob(t, factory.EmptyData)
	err := jp.DoWork(qj)
	test.AssertNotError(t, err, "")
	aj, err := archived_jobs.Get(qj.ID)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, aj.Status, models.StatusFailed)
}
