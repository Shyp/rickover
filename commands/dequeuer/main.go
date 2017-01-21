// Command dequeuer dequeues jobs and sends them to a downstream server.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Shyp/go-simple-metrics"
	"github.com/Shyp/rickover/config"
	"github.com/Shyp/rickover/dequeuer"
	"github.com/Shyp/rickover/models/db"
	"github.com/Shyp/rickover/services"
	"github.com/Shyp/rickover/setup"
	"golang.org/x/sync/errgroup"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g, _ := errgroup.WithContext(ctx)
	for _, p := range pools {
		if p != nil {
			p := p
			g.Go(func() error {
				err = p.Shutdown()
				if err != nil {
					log.Printf("Error shutting down pool: %s\n", err.Error())
				}
				return err
			})
		}
	}
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("All pools shut down. Quitting.")
}
