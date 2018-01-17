package models

import (
	"encoding/json"
	"time"

	"github.com/Shyp/go-types"
)

// An ArchivedJob is an in-memory representation of an archived job.
//
// ArchivedJob records are immutable, once they are stored in the database,
// they are never written, updated, or moved back to the jobs table.
type ArchivedJob struct {
	ID        types.PrefixUUID `json:"id"`
	Name      string           `json:"name"`
	Attempts  uint8            `json:"attempts"`
	Status    JobStatus        `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	Data      json.RawMessage  `json:"data"`
	ExpiresAt types.NullTime   `json:"expires_at"`
}
