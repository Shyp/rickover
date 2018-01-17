// Logic for interacting with the "archived_jobs" table.
package archived_jobs

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
	"github.com/Shyp/rickover/models/queued_jobs"
)

const Prefix = "job_"

// ErrNotFound indicates that the archived job was not found.
var ErrNotFound = errors.New("Archived job not found")

var createStmt *sql.Stmt
var getStmt *sql.Stmt

// Setup prepares all database statements.
func Setup() (err error) {
	if !db.Connected() {
		return errors.New("No DB connection was established, can't query")
	}

	if createStmt != nil {
		return
	}

	query := fmt.Sprintf(`-- archived_jobs.Create
INSERT INTO archived_jobs (%s) 
SELECT id, $2, $4, $3, data, expires_at
FROM queued_jobs 
WHERE id=$1
AND name=$2
RETURNING %s`, insertFields(), fields())
	createStmt, err = db.Conn.Prepare(query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf(`-- archived_jobs.Get
SELECT %s
FROM archived_jobs
WHERE id = $1`, fields())
	getStmt, err = db.Conn.Prepare(query)
	return
}

// Create an archived job with the given id, status, and attempts. Assumes that
// the job already exists in the queued_jobs table; the `data` field is copied
// from there. If the job does not exist, queued_jobs.ErrNotFound is returned.
func Create(id types.PrefixUUID, name string, status models.JobStatus, attempt uint8) (*models.ArchivedJob, error) {
	aj := new(models.ArchivedJob)
	var bt []byte
	err := createStmt.QueryRow(id, name, status, attempt).Scan(args(aj, &bt)...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, queued_jobs.ErrNotFound
		}
		err = dberror.GetError(err)
		return nil, err
	}
	aj.Data = json.RawMessage(bt)
	return aj, nil
}

// Get returns the archived job with the given id, or sql.ErrNoRows if it's
// not present.
func Get(id types.PrefixUUID) (*models.ArchivedJob, error) {
	if id.UUID == nil {
		return nil, errors.New("Invalid id")
	}
	aj := new(models.ArchivedJob)
	var bt []byte
	err := getStmt.QueryRow(id).Scan(args(aj, &bt)...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		err = dberror.GetError(err)
		return nil, err
	}
	aj.Data = json.RawMessage(bt)
	return aj, nil
}

// GetRetry attempts to retrieve the job attempts times before giving up.
func GetRetry(id types.PrefixUUID, attempts uint8) (job *models.ArchivedJob, err error) {
	for i := uint8(0); i < attempts; i++ {
		job, err = Get(id)
		if err == nil || err == ErrNotFound {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return
}

func insertFields() string {
	return `id,
	name,
	attempts,
	status,
	data,
	expires_at`
}

func fields() string {
	return fmt.Sprintf(`'%s' || id,
	name,
	attempts,
	status,
	data,
	created_at,
	expires_at`, Prefix)
}

func args(aj *models.ArchivedJob, byteptr *[]byte) []interface{} {
	return []interface{}{
		&aj.ID,
		&aj.Name,
		&aj.Attempts,
		&aj.Status,
		// can't scan into Data because of https://github.com/golang/go/issues/13905
		byteptr,
		&aj.CreatedAt,
		&aj.ExpiresAt,
	}
}
