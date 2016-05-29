package dequeuer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-types"
	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/nu7hatch/gouuid"
	"github.com/Shyp/rickover/dequeuer"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/test"
	"github.com/Shyp/rickover/test/db"
	"github.com/Shyp/rickover/test/factory"
)

func TestWorkerShutsDown(t *testing.T) {
	t.Skip("initial AddDequeuer sleep fails this test")
	db.SetUp(t)
	pool := dequeuer.NewPool("echo")
	for i := 0; i < 3; i++ {
		pool.AddDequeuer(factory.Processor("http://example.com"))
	}
	c1 := make(chan bool, 1)
	go func() {
		err := pool.Shutdown()
		test.AssertNotError(t, err, "")
		c1 <- true
	}()
	for {
		select {
		case <-c1:
			return
		case <-time.After(300 * time.Millisecond):
			t.Fatalf("pool did not shut down in 300ms")
		}
	}
}

// 1. Create a job type
// 2. Enqueue a job
// 3. Create a test server that replies with a 202
// 4. Ensure that the correct request is made to the server
func TestWorkerMakesCorrectRequest(t *testing.T) {
	t.Skip("initial AddDequeuer sleep fails this test")
	db.SetUp(t)
	defer db.TearDown(t)
	_, err := jobs.Create(factory.SampleJob)
	test.AssertNotError(t, err, "")

	pid := factory.RandomId("job_")

	var data json.RawMessage
	data, err = json.Marshal(factory.RD)
	test.AssertNotError(t, err, "")
	_, err = queued_jobs.Enqueue(pid, "echo", time.Now(), types.NullTime{Valid: false}, data)
	test.AssertNotError(t, err, "")

	// Capture the incoming http request - maybe pull this specific testing out
	c1 := make(chan bool, 1)
	var path, method, user string
	var ok bool
	var workRequest struct {
		Data     *factory.RandomData `json:"data"`
		Attempts uint8               `json:"attempts"`
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("received request")
		fmt.Println(r.URL)
		path = r.URL.Path
		method = r.Method
		user, _, ok = r.BasicAuth()
		json.NewDecoder(r.Body).Decode(&workRequest)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
		c1 <- true
	}))
	defer s.Close()
	jp := factory.Processor(s.URL)
	pool := dequeuer.NewPool("echo")
	pool.AddDequeuer(jp)
	defer pool.Shutdown()
	select {
	case <-c1:
		test.AssertEquals(t, path, fmt.Sprintf("/v1/jobs/echo/%s", pid.String()))
		test.AssertEquals(t, method, "POST")
		test.AssertEquals(t, ok, true)
		test.AssertEquals(t, user, "jobs")
		test.AssertDeepEquals(t, workRequest.Data, factory.RD)
		test.AssertEquals(t, workRequest.Attempts, uint8(3))
		return
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Server did not receive a request in 200ms, quitting")
	}
}

// 1. Create a job type
// 2. Enqueue a job
// 2a. Create twenty worker nodes
// 3. Create a test server that replies with a 202
// 4. Ensure that only one request is made to the server
func TestWorkerMakesExactlyOneRequest(t *testing.T) {
	t.Skip("implement the success/failure callbacks")
	db.SetUp(t)
	defer db.TearDown(t)
	_, err := jobs.Create(factory.SampleJob)
	test.AssertNotError(t, err, "")

	id, _ := uuid.NewV4()
	pid, _ := types.NewPrefixUUID(fmt.Sprintf("job_%s", id))

	var data json.RawMessage
	data, err = json.Marshal(factory.RD)
	test.AssertNotError(t, err, "")
	_, err = queued_jobs.Enqueue(pid, "echo", time.Now(), types.NullTime{Valid: false}, data)
	test.AssertNotError(t, err, "")

	c1 := make(chan bool, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
		c1 <- true
	}))
	defer s.Close()
	pool := dequeuer.NewPool("echo")
	for i := 0; i < 20; i++ {
		jp := factory.Processor(s.URL)
		pool.AddDequeuer(jp)
	}
	defer pool.Shutdown()
	count := 0
	for {
		select {
		case <-c1:
			count++
		case <-time.After(100 * time.Millisecond):
			test.AssertEquals(t, count, 1)
			return
		}
	}
}
