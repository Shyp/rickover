// Dequeue jobs.
package main

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

func main() {
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

	// We're going to make a lot of requests to the same downstream service.
	httpConns, err := config.GetInt("HTTP_MAX_IDLE_CONNS")
	if err == nil {
		config.SetMaxIdleConnsPerHost(httpConns)
	} else {
		config.SetMaxIdleConnsPerHost(100)
	}

	metrics.Namespace = "rickover.dequeuer"
	metrics.Start("worker")

	downstreamPassword := os.Getenv("DOWNSTREAM_WORKER_AUTH")
	if downstreamPassword == "" {
		log.Printf("No DOWNSTREAM_WORKER_AUTH configured, setting an empty password for auth")
	}

	parsedUrl := config.GetURLOrBail("DOWNSTREAM_URL")
	jp := services.NewJobProcessor(parsedUrl.String(), downstreamPassword)

	// This creates a pool of dequeuers and starts them.
	pools, err := dequeuer.CreatePools(jp, 200*time.Millisecond)
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
