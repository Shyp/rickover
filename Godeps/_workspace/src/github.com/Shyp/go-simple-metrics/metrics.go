// The metrics package instruments your code.
//
// Set DEBUG=metrics environment variable to print metrics to stdout.
package metrics

import (
	"fmt"
	"log"
	"os"
	"time"

	godebug "github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-debug"
	librato "github.com/Shyp/rickover/Godeps/_workspace/src/github.com/mihasya/go-metrics-librato"
	metrics "github.com/Shyp/rickover/Godeps/_workspace/src/github.com/rcrowley/go-metrics"
)

var debug = godebug.Debug("metrics")

// Namespace is the namespace under which all metrics will get incremented.
// Typically this should match up with the running service ("api", "admin",
// "jobs", "parcels", &c).
var Namespace string

func getWithNamespace(metricName string) string {
	if Namespace == "" {
		return metricName
	} else {
		return fmt.Sprintf("%s.%s", Namespace, metricName)
	}
}

// Start initializes the metrics client. You must call this before sending
// metrics, or metrics will not get published to Librato.
func Start(source string) {
	token := os.Getenv("LIBRATO_TOKEN")
	if token == "" {
		log.Printf("Could not find LIBRATO_TOKEN environment variable; no metrics will be logged")
	} else {
		go librato.Librato(
			metrics.DefaultRegistry,
			10*time.Second,
			"devops@shyp.com",
			token,
			source,
			[]float64{0.5, 0.95, 0.99, 1},
			time.Millisecond,
		)
	}
}

// Increment a counter with the given name.
func Increment(name string) {
	mn := getWithNamespace(name)
	m := metrics.GetOrRegisterMeter(mn, nil)
	m.Mark(1)
	debug("increment %s 1", name)
}

// Measure that the given metric has the given value.
func Measure(name string, value int64) {
	mn := getWithNamespace(name)
	g := metrics.GetOrRegisterGauge(mn, nil)
	g.Update(value)
	debug("measure %s %d", name, value)
}

// Add a new timing measurement for the given metric.
func Time(name string, value time.Duration) {
	mn := getWithNamespace(name)
	t := metrics.GetOrRegisterTimer(mn, nil)
	t.Update(value)
	debug("time %s %v", name, value)
}
