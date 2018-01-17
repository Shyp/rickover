// Package setup helps initialize the database and all queries.
package setup

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Shyp/go-simple-metrics"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/archived_jobs"
	"github.com/Shyp/rickover/models/db"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/models/queued_jobs"
)

var mu sync.Mutex

// TODO not sure for the best place for this to live.
var activeQueriesStmt *sql.Stmt

func prepare() (err error) {
	if !db.Connected() {
		return errors.New("No DB connection was established, can't query")
	}

	activeQueriesStmt, err = db.Conn.Prepare(`-- setup.GetActiveQueries
SELECT count(*) FROM pg_stat_activity 
WHERE state='active'
	`)
	return
}

func GetActiveQueries() (count int64, err error) {
	err = activeQueriesStmt.QueryRow().Scan(&count)
	return
}

// TODO all of these should use a different database connection than the server
// or the worker, to avoid contention.
func MeasureActiveQueries(interval time.Duration) {
	for _ = range time.Tick(interval) {
		count, err := GetActiveQueries()
		if err == nil {
			go metrics.Measure("active_queries.count", count)
		} else {
			go metrics.Increment("active_queries.error")
		}
	}
}

func MeasureQueueDepth(interval time.Duration) {
	for _ = range time.Tick(interval) {
		allCount, readyCount, err := queued_jobs.CountReadyAndAll()
		if err == nil {
			go metrics.Measure("queue_depth.all", int64(allCount))
			go metrics.Measure("queue_depth.ready", int64(readyCount))
		} else {
			go metrics.Increment("queue_depth.error")
		}
	}
}

func MeasureInProgressJobs(interval time.Duration) {
	for _ = range time.Tick(interval) {
		m, err := queued_jobs.GetCountsByStatus(models.StatusInProgress)
		if err == nil {
			count := int64(0)
			for k, v := range m {
				count += v
				go metrics.Measure(fmt.Sprintf("queued_jobs.%s.in_progress", k), v)
			}
			go metrics.Measure("queued_jobs.in_progress", count)
		} else {
			go metrics.Increment("queued_jobs.in_progress.error")
		}
	}
}

// DB initializes a connection to the database, and prepares queries on all
// models.
func DB(connector db.Connector, dbConns int) error {
	mu.Lock()
	defer mu.Unlock()
	if db.Conn != nil {
		if err := db.Conn.Ping(); err == nil {
			// Already connected.
			return nil
		}
	}
	conn, err := connector.Connect(dbConns)
	db.Conn = conn
	if err != nil {
		return errors.New("Could not establish a database connection: " + err.Error())
	}
	if err := db.Conn.Ping(); err != nil {
		return errors.New("Could not establish a database connection: " + err.Error())
	}
	return PrepareAll()
}

func PrepareAll() error {
	if err := jobs.Setup(); err != nil {
		return err
	}
	if err := queued_jobs.Setup(); err != nil {
		return err
	}
	if err := archived_jobs.Setup(); err != nil {
		return err
	}
	if err := prepare(); err != nil {
		return err
	}
	return nil
}
