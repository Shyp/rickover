package metrics

import (
	"testing"
	"time"

	"github.com/Shyp/goshyp/test"
	"github.com/rcrowley/go-metrics"
)

func TestNamespace(t *testing.T) {
	originalNamespace := Namespace
	defer func() {
		Namespace = originalNamespace
	}()
	Namespace = "foo"
	if getWithNamespace("bar") != "foo.bar" {
		t.Errorf("expected getWithNamespace(bar) to be foo.bar, was %s", getWithNamespace("bar"))
	}

	Namespace = ""
	test.AssertEquals(t, getWithNamespace("bar"), "bar")
}

func TestIncrementIncrements(t *testing.T) {
	Increment("bar")
	Increment("bar")
	Increment("bar")
	mn := metrics.GetOrRegisterCounter("bar", nil)
	if mn.Count() != 3 {
		t.Errorf("expected Count() to be 3, was %d", mn.Count())
	}
}

func ExampleIncrement() {
	Start("web")
	Increment("dequeue.success")
}

func ExampleMeasure() {
	Start("web")
	Measure("workers.active", 6)
}

func ExampleTime() {
	Start("web")
	start := time.Now()
	time.Sleep(3)
	Time("auth.latency", time.Since(start))
}
