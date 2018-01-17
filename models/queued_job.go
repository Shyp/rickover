package models

import (
	"encoding/json"
	"time"

	"github.com/Shyp/go-types"
)

type JobStatus string

// StatusQueued indicates a QueuedJob is scheduled to be run at some point in
// the future.
const StatusQueued = JobStatus("queued")

// StatusInProgress indicates a QueuedJob has been dequeued, and is being
// worked on.
const StatusInProgress = JobStatus("in-progress")

// A QueuedJob is a job to be run at a point in the future.
//
// QueuedJobs can have the status "queued" (to be run at some point), or
// "in-progress" (a dequeuer is acting on them).
type QueuedJob struct {
	ID        types.PrefixUUID `json:"id"`
	Name      string           `json:"name"`
	Attempts  uint8            `json:"attempts"`
	RunAfter  time.Time        `json:"run_after"`
	ExpiresAt types.NullTime   `json:"expires_at"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Status    JobStatus        `json:"status"`
	Data      json.RawMessage  `json:"data"`
}
