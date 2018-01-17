// Logic for interacting with the "queued_jobs" table.
package queued_jobs

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Shyp/go-dberror"
	"github.com/Shyp/go-types"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/db"
)

const Prefix = "job_"

// ErrNotFound indicates that the job was not found.
var ErrNotFound = errors.New("Queued job not found")

// UnknownOrArchivedError is raised when the job type is unknown or the job has
// already been archived. It's unfortunate we can't distinguish these, but more
// important to minimize the total number of queries to the database.
type UnknownOrArchivedError struct {
	Err string
}

func (e *UnknownOrArchivedError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Err
}

var enqueueStmt *sql.Stmt
var getStmt *sql.Stmt
var deleteStmt *sql.Stmt
var acquireStmt *sql.Stmt
var decrementStmt *sql.Stmt
var countReadyAndAllStmt *sql.Stmt
var countsByStatusStmt *sql.Stmt
var oldJobsStmt *sql.Stmt

// StuckJobLimit is the maximum number of stuck jobs to fetch in one database
// query.
var StuckJobLimit = 100

func Setup() (err error) {
	if !db.Connected() {
		return errors.New("No DB connection was established, can't query")
	}

	if enqueueStmt != nil {
		return
	}

	query := fmt.Sprintf(`-- queued_jobs.Enqueue
INSERT INTO queued_jobs (%s) 
SELECT $1, name, attempts, $3, $4, '%s', $5
FROM jobs 
WHERE name=$2
AND NOT EXISTS (
	SELECT id FROM archived_jobs WHERE id=$1
)
RETURNING %s`, insertFields(), models.StatusQueued, fields())
	enqueueStmt, err = db.Conn.Prepare(query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf(`-- queued_jobs.Get
SELECT %s
FROM queued_jobs
WHERE id = $1`, fields())
	getStmt, err = db.Conn.Prepare(query)
	if err != nil {
		return err
	}

	query = `-- queued_jobs.Delete
	DELETE FROM queued_jobs WHERE id = $1`
	deleteStmt, err = db.Conn.Prepare(query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf(`-- queued_jobs.Acquire
WITH queued_job as (
	SELECT id AS inner_id
	FROM queued_jobs
	WHERE status='%[1]s'
		AND name = $1
		AND run_after <= now()
	ORDER BY created_at ASC
	LIMIT 1
	FOR UPDATE
) UPDATE queued_jobs
SET status='%[2]s',
	updated_at=now()
FROM queued_job
WHERE queued_jobs.id = queued_job.inner_id 
	AND status='%[1]s'
RETURNING %[3]s`, models.StatusQueued, models.StatusInProgress, fields())
	acquireStmt, err = db.Conn.Prepare(query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf(`-- queued_jobs.Decrement
UPDATE queued_jobs
SET status = '%s',
	updated_at = now(),
	attempts = attempts - 1,
	run_after = $3
WHERE id = $1
	AND attempts=$2
	RETURNING %s`, models.StatusQueued, fields())
	decrementStmt, err = db.Conn.Prepare(query)
	if err != nil {
		return err
	}

	query = `-- queued_jobs.CountReadyAndAll
WITH all_count AS (
	SELECT count(*) FROM queued_jobs
), ready_count AS (
	SELECT count(*) FROM queued_jobs WHERE run_after <= now()
) 
SELECT all_count.count, ready_count.count 
FROM all_count, ready_count`
	countReadyAndAllStmt, err = db.Conn.Prepare(query)
	if err != nil {
		return err
	}

	query = `-- queued_jobs.GetCountsByStatus
SELECT name, count(*) FROM queued_jobs WHERE status=$1 GROUP BY name`
	countsByStatusStmt, err = db.Conn.Prepare(query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf(`-- queued_jobs.GetOldInProgressJobs
SELECT %s FROM queued_jobs WHERE status='%s' AND updated_at < $1 LIMIT %d`,
		fields(), models.StatusInProgress, StuckJobLimit)
	oldJobsStmt, err = db.Conn.Prepare(query)
	if err != nil {
		return err
	}
	return
}

// Enqueue creates a new queued job with the given ID and fields. A
// dberror.Error will be returned if Postgres returns a constraint failure -
// job exists, job name unknown, &c. A sql.ErrNoRows will be returned if the
// `name` does not exist in the jobs table. Otherwise the QueuedJob will be
// returned.
func Enqueue(id types.PrefixUUID, name string, runAfter time.Time, expiresAt types.NullTime, data json.RawMessage) (*models.QueuedJob, error) {
	qj := new(models.QueuedJob)
	// need to scan into a []byte, https://github.com/golang/go/issues/13905
	var bt []byte
	err := enqueueStmt.QueryRow(id, name, runAfter, expiresAt, []byte(data)).Scan(args(qj, &bt)...)
	if err != nil {
		if err == sql.ErrNoRows {
			e := &UnknownOrArchivedError{
				Err: fmt.Sprintf("Job type %s does not exist or the job with that id has already been archived", name),
			}
			return nil, e
		}
		return nil, dberror.GetError(err)
	}
	qj.Data = json.RawMessage(bt)
	return qj, err
}

// Get the queued job with the given id. Returns the job, or an error. If no
// record could be found, the error will be `queued_jobs.ErrNotFound`.
func Get(id types.PrefixUUID) (*models.QueuedJob, error) {
	if id.UUID == nil {
		return nil, errors.New("Invalid id")
	}
	qj := new(models.QueuedJob)
	var bt []byte
	err := getStmt.QueryRow(id).Scan(args(qj, &bt)...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, dberror.GetError(err)
	}
	qj.Data = json.RawMessage(bt)
	return qj, nil
}

// GetRetry attempts to retrieve the job attempts times before giving up.
func GetRetry(id types.PrefixUUID, attempts uint8) (job *models.QueuedJob, err error) {
	for i := uint8(0); i < attempts; i++ {
		job, err = Get(id)
		if err == nil || err == ErrNotFound {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return
}

// Delete deletes the given queued job. Returns nil if the job was deleted
// successfully. If no job exists to be deleted, sql.ErrNoRows is returned.
func Delete(id types.PrefixUUID) error {
	if id.UUID == nil {
		return errors.New("Invalid id")
	}
	res, err := deleteStmt.Exec(id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	} else if rows == 1 {
		return nil
	} else {
		// This should not be possible because of database constraints
		return fmt.Errorf("Multiple rows (%d) deleted for job %s, please investigate", rows, id)
	}
}

// DeleteRetry attempts to Delete the item `attempts` times.
func DeleteRetry(id types.PrefixUUID, attempts uint8) error {
	for i := uint8(0); i < attempts; i++ {
		err := Delete(id)
		if err == nil || err == ErrNotFound {
			return err
		}
	}
	return nil
}

// Acquire a queued job with the given name that's able to run now. Returns
// the queued job and a boolean indicating whether the SELECT query found
// a row, or a generic error/sql.ErrNoRows if no jobs are available.
func Acquire(name string) (*models.QueuedJob, error) {
	qj := new(models.QueuedJob)
	var bt []byte

	rows, err := acquireStmt.Query(name)
	if err != nil {
		err = dberror.GetError(err)
		return nil, err
	}
	defer rows.Close()
	count := 0
	scanned := false
	for rows.Next() {
		count += 1
		if !scanned {
			rows.Scan(args(qj, &bt)...)
			scanned = true
		}
	}
	if count == 0 {
		return nil, sql.ErrNoRows
	}
	if count > 1 {
		fmt.Println(time.Now().UTC())
		panic(fmt.Sprintf("Too many rows affected by Acquire for '%s': %d", name, count))
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	qj.Data = json.RawMessage(bt)
	return qj, nil
}

// Decrement decrements the attempts counter for an existing job, and sets
// its status back to 'queued'. If the queued job does not exist, or the
// attempts counter in the database does not match the passed in attempts
// value, sql.ErrNoRows will be returned.
//
// attempts: The current value of the `attempts` column, the returned attempts
// value will be this number minus 1.
func Decrement(id types.PrefixUUID, attempts uint8, runAfter time.Time) (*models.QueuedJob, error) {
	qj := new(models.QueuedJob)
	var bt []byte
	err := decrementStmt.QueryRow(id, attempts, runAfter).Scan(args(qj, &bt)...)
	if err != nil {
		err = dberror.GetError(err)
		return nil, err
	}
	qj.Data = json.RawMessage(bt)
	return qj, nil
}

// GetOldInProgressJobs finds queued in-progress jobs with an updated_at
// timestamp older than olderThan. A maximum of StuckJobLimit jobs will be
// returned.
func GetOldInProgressJobs(olderThan time.Time) ([]*models.QueuedJob, error) {
	rows, err := oldJobsStmt.Query(olderThan)
	var jobs []*models.QueuedJob
	if err != nil {
		return jobs, err
	}
	defer rows.Close()
	for rows.Next() {
		qj := new(models.QueuedJob)
		var bt []byte
		err = rows.Scan(args(qj, &bt)...)
		if err != nil {
			return jobs, err
		}
		qj.Data = json.RawMessage(bt)
		jobs = append(jobs, qj)
	}
	err = rows.Err()
	return jobs, err
}

// CountReadyAndAll returns the total number of queued and ready jobs in the
// table.
func CountReadyAndAll() (allCount int, readyCount int, err error) {
	err = countReadyAndAllStmt.QueryRow().Scan(&allCount, &readyCount)
	return
}

// GetCountsByStatus returns a map with each job type as the key, followed by
// the number of <status> jobs it has. For example:
//
// "echo": 5,
// "remind-assigned-driver": 7,
func GetCountsByStatus(status models.JobStatus) (map[string]int64, error) {
	rows, err := countsByStatusStmt.Query(status)
	m := make(map[string]int64)
	if err != nil {
		return m, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var count int64
		err = rows.Scan(&name, &count)
		if err != nil {
			return m, err
		}
		m[name] = count
	}
	err = rows.Err()
	return m, err
}

func insertFields() string {
	return `id,
	name,
	attempts,
	run_after,
	expires_at,
	status,
	data`
}

func fields() string {
	return fmt.Sprintf(`'%s' || id,
	name,
	attempts,
	run_after,
	expires_at,
	status,
	data,
	created_at,
	updated_at`, Prefix)
}

func args(qj *models.QueuedJob, byteptr *[]byte) []interface{} {
	return []interface{}{
		&qj.ID,
		&qj.Name,
		&qj.Attempts,
		&qj.RunAfter,
		&qj.ExpiresAt,
		&qj.Status,
		// can't scan into Data because of https://github.com/golang/go/issues/13905
		byteptr,
		&qj.CreatedAt,
		&qj.UpdatedAt,
	}
}
