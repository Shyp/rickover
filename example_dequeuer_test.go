// Run the rickover dequeuer. Configure the following environment variables:
//
// DATABASE_URL: Postgres connection string (see Makefile)
// PG_WORKER_POOL_SIZE: Maximum number of database connections from this process
// DOWNSTREAM_URL: Downstream server that can perform the work
// DOWNSTREAM_WORKER_AUTH: Basic Auth password for downstream server (user "jobs")
//
// Create job types by making a POST request to /v1/jobs with the job name and
// concurrency. After that, CreatePools will start and run dequeuers for those
// types.

package rickover

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Shyp/go-simple-metrics"
	"github.com/Shyp/rickover/config"
	"github.com/Shyp/rickover/dequeuer"
	"github.com/Shyp/rickover/models/db"
	"github.com/Shyp/rickover/services"
	"github.com/Shyp/rickover/setup"
)

var dbConns int
var downstreamUrl string
var downstreamPassword string

func init() {
	var err error
	dbConns, err = config.GetInt("PG_WORKER_POOL_SIZE")
	if err != nil {
		log.Printf("Error getting database pool size: %s. Defaulting to 20", err)
		dbConns = 20
	}

	downstreamPassword = os.Getenv("DOWNSTREAM_WORKER_AUTH")

	metrics.Namespace = "rickover.dequeuer"
}

func Example_dequeuer() {
	if err := setup.DB(db.DefaultConnection, dbConns); err != nil {
		log.Fatal(err)
	}

	metrics.Start("worker")

	go setup.MeasureActiveQueries(1 * time.Second)
	go setup.MeasureQueueDepth(5 * time.Second)
	go setup.MeasureInProgressJobs(1 * time.Second)

	// Every minute, check for in-progress jobs that haven't been updated for
	// 7 minutes, and mark them as failed.
	go services.WatchStuckJobs(1*time.Minute, 7*time.Minute)

	downstreamUrl = config.GetURLOrBail("DOWNSTREAM_URL").String()
	jp := services.NewJobProcessor(downstreamUrl, downstreamPassword)

	// CreatePools will read all job types out of the jobs table, then start
	// all dequeuers for those jobs.
	pools, err := dequeuer.CreatePools(jp, 200*time.Millisecond)
	if err != nil {
		log.Fatal(err)
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	sig := <-sigterm
	fmt.Printf("Caught signal %v, shutting down...\n", sig)
	var wg sync.WaitGroup
	for _, p := range pools {
		if p != nil {
			wg.Add(1)
			go func(p *dequeuer.Pool) {
				err = p.Shutdown()
				if err != nil {
					log.Printf("Error shutting down pool: %s\n", err.Error())
				}
				wg.Done()
			}(p)
		}
	}
	wg.Wait()
	fmt.Println("All pools shut down. Quitting.")
}
