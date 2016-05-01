package dequeuer

import (
	"testing"
	"time"

	"github.com/Shyp/rickover/dequeuer"
	"github.com/Shyp/rickover/test"
	"github.com/Shyp/rickover/test/db"
	"github.com/Shyp/rickover/test/factory"
)

func TestWorkerShutsDown(t *testing.T) {
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
