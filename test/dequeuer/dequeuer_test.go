package dequeuer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Shyp/rickover/dequeuer"
	"github.com/Shyp/rickover/test"
	"github.com/Shyp/rickover/test/factory"
)

func TestAll(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	t.Run("Parallel", func(t *testing.T) {
		t.Run("TestWorkerShutsDown", testWorkerShutsDown)
		t.Run("TestWorkerMakesCorrectRequest", testWorkerMakesCorrectRequest)
		t.Run("TestWorkerMakesExactlyOneRequest", testWorkerMakesCorrectRequest)
	})
}

func testWorkerShutsDown(t *testing.T) {
	t.Parallel()
	poolname := factory.RandomId("pool")
	pool := dequeuer.NewPool(poolname.String())
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
func testWorkerMakesCorrectRequest(t *testing.T) {
	t.Parallel()
	qj := factory.CreateQJ(t)

	c1 := make(chan bool, 1)
	var path, method, user string
	var ok bool
	var workRequest struct {
		Data     *factory.RandomData `json:"data"`
		Attempts uint8               `json:"attempts"`
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		user, _, ok = r.BasicAuth()
		err := json.NewDecoder(r.Body).Decode(&workRequest)
		test.AssertNotError(t, err, "decoding request body")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
		c1 <- true
		close(c1)
	}))
	defer s.Close()
	jp := factory.Processor(s.URL)
	pool := dequeuer.NewPool(qj.Name)
	pool.AddDequeuer(jp)
	defer pool.Shutdown()
	select {
	case <-c1:
		test.AssertEquals(t, path, fmt.Sprintf("/v1/jobs/%s/%s", qj.Name, qj.ID.String()))
		test.AssertEquals(t, method, "POST")
		test.AssertEquals(t, ok, true)
		test.AssertEquals(t, user, "jobs")
		test.AssertDeepEquals(t, workRequest.Data, factory.RD)
		test.AssertEquals(t, workRequest.Attempts, qj.Attempts)
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
func testWorkerMakesExactlyOneRequest(t *testing.T) {
	t.Parallel()
	qj := factory.CreateQJ(t)

	c1 := make(chan bool, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
		c1 <- true
	}))
	defer s.Close()
	pool := dequeuer.NewPool(qj.Name)
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

func TestCreatePools(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	qj := factory.CreateQJ(t)
	factory.CreateQJ(t)
	proc := factory.Processor("http://example.com")
	pools, err := dequeuer.CreatePools(proc, 0)
	test.AssertNotError(t, err, "CreatePools")
	test.AssertEquals(t, len(pools), 2)
	foundPool := false
	for _, pool := range pools {
		if pool.Name == qj.Name {
			foundPool = true
			test.AssertEquals(t, len(pool.Dequeuers), 3)
			for i, dq := range pool.Dequeuers {
				test.AssertEquals(t, dq.ID, i+1)
				test.AssertEquals(t, dq.W, proc)
			}
		}
	}
	test.Assert(t, foundPool, "Didn't create a pool for the job type")
}
