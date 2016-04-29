// Setup helps initialize applications.
package setup

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-simple-metrics"
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

// DefaultConnection connects to a Postgres database using the DATABASE_URL
// environment variable.
var DefaultConnection = &DatabaseURLConnector{}

// DatabaseURLConnector connects to the database using the DATABASE_URL
// environment variable.
type DatabaseURLConnector struct {
	mu sync.Mutex
}

// Connect to the database using the DATABASE_URL environment variable with the
// given number of database connections, and store the result in conn.
func (dc *DatabaseURLConnector) Connect(conn *sql.DB, dbConns int) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if conn == nil {
		return errors.New("setup: Cannot assign to nil conn")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("setup: No value provided for DATABASE_URL, cannot connect")
	}
	d, err := sql.Open("postgres", url)
	if err != nil {
		return err
	}
	d.SetMaxOpenConns(dbConns)
	if dbConns > 100 {
		d.SetMaxIdleConns(dbConns - 20)
	} else if dbConns > 50 {
		d.SetMaxIdleConns(dbConns - 10)
	} else if dbConns > 10 {
		d.SetMaxIdleConns(dbConns - 3)
	} else if dbConns > 5 {
		d.SetMaxIdleConns(dbConns - 2)
	}
	*conn = *d
	return nil
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
	if db.Conn == nil {
		db.Conn = &sql.DB{}
	} else {
		if err := db.Conn.Ping(); err == nil {
			// Already connected.
			return nil
		}
	}
	if err := connector.Connect(db.Conn, dbConns); err != nil {
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
