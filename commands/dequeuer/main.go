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

	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-simple-metrics"
	"github.com/Shyp/rickover/config"
	"github.com/Shyp/rickover/dequeuer"
	"github.com/Shyp/rickover/downstream"
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

	err = setup.DB(setup.DefaultConnection, dbConns)
	checkError(err)

	go setup.MeasureActiveQueries(1 * time.Second)
	go setup.MeasureQueueDepth(5 * time.Second)
	go setup.MeasureInProgressJobs(1 * time.Second)

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
	jp := &services.JobProcessor{
		Client:  downstream.NewClient("jobs", downstreamPassword, parsedUrl.String()),
		Timeout: 5 * time.Minute,
	}

	// This creates a pool of dequeuers and starts them.
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
