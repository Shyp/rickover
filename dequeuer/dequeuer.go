// Package dequeuer retrieves jobs from the database and does some work.
package dequeuer

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/Shyp/go-dberror"
	"github.com/Shyp/go-simple-metrics"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
	"golang.org/x/sync/errgroup"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewPool(name string) *Pool {
	return &Pool{
		Name: name,
	}
}

type Pools []*Pool

// NumDequeuers returns the total number of dequeuers across all pools.
func (ps Pools) NumDequeuers() int {
	dequeuerCount := 0
	for _, pool := range ps {
		dequeuerCount = dequeuerCount + len(pool.Dequeuers)
	}
	return dequeuerCount
}

// CreatePools creates job pools for all jobs in the database. The provided
// Worker w will be shared between all dequeuers, so it must be thread safe.
func CreatePools(w Worker, maxInitialJitter time.Duration) (Pools, error) {
	jobs, err := jobs.GetAll()
	if err != nil {
		return Pools{}, err
	}

	pools := make([]*Pool, len(jobs))
	var g errgroup.Group
	for i, job := range jobs {
		// Copy these so we don't have a concurrency/race problem when the
		// counter iterates
		i := i
		name := job.Name
		concurrency := job.Concurrency
		g.Go(func() error {
			p := NewPool(name)
			var innerg errgroup.Group
			for j := uint8(0); j < concurrency; j++ {
				innerg.Go(func() error {
					time.Sleep(time.Duration(rand.Float64()) * maxInitialJitter)
					err := p.AddDequeuer(w)
					if err != nil {
						log.Print(err)
					}
					return err
				})
			}
			if err := innerg.Wait(); err != nil {
				return err
			}
			pools[i] = p
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return pools, nil
}

// A Pool contains an array of dequeuers, all of which perform work for the
// same models.Job.
type Pool struct {
	Dequeuers              []*Dequeuer
	Name                   string
	receivedShutdownSignal bool
	mu                     sync.Mutex
	wg                     sync.WaitGroup
}

type Dequeuer struct {
	ID       int
	QuitChan chan bool
	W        Worker
}

// A Worker does some work with a QueuedJob. Worker implementations may be
// shared and should be threadsafe.
type Worker interface {
	// DoWork does whatever work should be done with the queued
	// job. Success and failure for the job are marked by hitting
	// services.HandleStatusCallback, or POST /v1/jobs/:job-name/:job-id
	// (over HTTP).
	//
	// A good pattern is for DoWork to make a HTTP request to a downstream
	// service, and then for that service to make a HTTP callback to report
	// success or failure.
	//
	// If DoWork is unable to get the work to be done, it should call
	// HandleStatusCallback with a failed callback; errors are logged, but
	// otherwise nothing else is done with them.
	DoWork(*models.QueuedJob) error

	// Sleep returns the amount of time to sleep between failed attempts to
	// acquire a queued job. The default implementation sleeps for 20, 40, 80,
	// 160, ..., up to a maximum of 10 seconds between attempts.
	Sleep(failedAttempts uint32) time.Duration
}

// AddDequeuer adds a Dequeuer to the Pool. w should be the work that the
// Dequeuer will do with a dequeued job.
func (p *Pool) AddDequeuer(w Worker) error {
	if p.receivedShutdownSignal {
		return poolShutdown
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	d := &Dequeuer{
		ID:       len(p.Dequeuers) + 1,
		QuitChan: make(chan bool, 1),
		W:        w,
	}
	p.Dequeuers = append(p.Dequeuers, d)
	p.wg.Add(1)
	go d.Work(p.Name, &p.wg)
	return nil
}

var emptyPool = errors.New("No workers left to dequeue")
var poolShutdown = errors.New("Cannot add worker because the pool is shutting down")

// RemoveDequeuer removes a dequeuer from the pool and sends that dequeuer
// a shutdown signal.
func (p *Pool) RemoveDequeuer() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.Dequeuers) == 0 {
		return emptyPool
	}
	dq := p.Dequeuers[0]
	p.Dequeuers = append(p.Dequeuers[:0], p.Dequeuers[1:]...)
	dq.QuitChan <- true
	close(dq.QuitChan)
	return nil
}

// Shutdown all workers in the pool.
func (p *Pool) Shutdown() error {
	p.receivedShutdownSignal = true
	l := len(p.Dequeuers)
	for i := 0; i < l; i++ {
		err := p.RemoveDequeuer()
		if err != nil {
			return err
		}
	}
	p.wg.Wait()
	return nil
}

func (d *Dequeuer) Work(name string, wg *sync.WaitGroup) {
	defer wg.Done()
	failedAcquireCount := uint32(0)
	waitDuration := time.Duration(0)
	for {
		select {
		case <-d.QuitChan:
			log.Printf("%s worker %d quitting\n", name, d.ID)
			return

		case <-time.After(waitDuration):
			start := time.Now()
			qj, err := queued_jobs.Acquire(name)
			go metrics.Time("acquire.latency", time.Since(start))
			if err == nil {
				failedAcquireCount = 0
				waitDuration = time.Duration(0)
				err = d.W.DoWork(qj)
				if err != nil {
					log.Printf("worker: Error processing job %s: %s", qj.ID.String(), err)
					go metrics.Increment(fmt.Sprintf("dequeue.%s.error", name))
				} else {
					go metrics.Increment(fmt.Sprintf("dequeue.%s.success", name))
				}
			} else {
				dberr, ok := err.(*dberror.Error)
				if ok && dberr.Code == dberror.CodeLockNotAvailable {
					// SELECT 1 returned a record but another thread
					// got it. Don't sleep at all.
					go metrics.Increment(fmt.Sprintf("dequeue.%s.nowait", name))
					failedAcquireCount = 0
					waitDuration = time.Duration(0)
					continue
				}

				failedAcquireCount++
				waitDuration = d.W.Sleep(failedAcquireCount)
			}
		}
	}
}
