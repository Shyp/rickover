// Run the rickover dequeuer.
//
// All of the project defaults are used. There is one authenticated user for
// basic auth, the user is "test" and the password is "hymanrickover". You will
// want to copy this binary and add your own authentication scheme.

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

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Example_dequeuer() {
	dbConns, err := config.GetInt("PG_WORKER_POOL_SIZE")
	if err != nil {
		log.Printf("Error getting database pool size: %s. Defaulting to 20", err)
		dbConns = 20
	}

	err = setup.DB(db.DefaultConnection, dbConns)
	checkError(err)

	go setup.MeasureActiveQueries(1 * time.Second)
	go setup.MeasureQueueDepth(5 * time.Second)
	go setup.MeasureInProgressJobs(1 * time.Second)

	// Every minute, check for in-progress jobs that haven't been updated for
	// 7 minutes, and mark them as failed.
	go services.WatchStuckJobs(1*time.Minute, 7*time.Minute)

	metrics.Namespace = "rickover.dequeuer"
	metrics.Start("worker")

	downstreamUrl := config.GetURLOrBail("DOWNSTREAM_URL")
	downstreamPassword := os.Getenv("DOWNSTREAM_WORKER_AUTH")
	jp := services.NewJobProcessor(downstreamUrl.String(), downstreamPassword)

	// CreatePools will read all job types out of the jobs table, then start
	// all dequeuers for those jobs.
	pools, err := dequeuer.CreatePools(jp)
	checkError(err)

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
