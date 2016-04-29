package models

import (
	"encoding/json"
	"time"

	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-types"
)

type ArchivedJob struct {
	Id        types.PrefixUUID `json:"id"`
	Name      string           `json:"name"`
	Attempts  uint8            `json:"attempts"`
	Status    JobStatus        `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	Data      json.RawMessage  `json:"data"`
}
