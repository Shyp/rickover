package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-types"
)

type Job struct {
	Name             string           `json:"name"`
	DeliveryStrategy DeliveryStrategy `json:"delivery_strategy"`
	Attempts         uint8            `json:"attempts"`
	Concurrency      uint8            `json:"concurrency"`
	CreatedAt        time.Time        `json:"created_at"`
}

type QueuedJob struct {
	Id        types.PrefixUUID `json:"id"`
	Name      string           `json:"name"`
	Attempts  uint8            `json:"attempts"`
	RunAfter  time.Time        `json:"run_after"`
	ExpiresAt types.NullTime   `json:"expires_at"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Status    JobStatus        `json:"status"`
	Data      json.RawMessage  `json:"data"`
}
type DeliveryStrategy string

const StrategyAtLeastOnce = DeliveryStrategy("at_least_once")
const StrategyAtMostOnce = DeliveryStrategy("at_most_once")

func (d DeliveryStrategy) Value() (driver.Value, error) {
	return string(d), nil
}

type JobStatus string

const StatusQueued = JobStatus("queued")
const StatusInProgress = JobStatus("in-progress")
const StatusSucceeded = JobStatus("succeeded")
const StatusFailed = JobStatus("failed")
const StatusExpired = JobStatus("expired")

// Scan implements the Scanner interface.
func (j *JobStatus) Scan(src interface{}) error {
	if src == nil {
		return nil
	} else if txt, ok := src.(string); ok {
		*j = JobStatus(txt)
		return nil
	} else if txt, ok := src.([]byte); ok {
		*j = JobStatus(string(txt))
		return nil
	}
	return fmt.Errorf("Unsupported JobStatus: %#v", src)
}

func (j JobStatus) Value() (driver.Value, error) {
	return string(j), nil
}

// Scan implements the Scanner interface.
func (d *DeliveryStrategy) Scan(src interface{}) error {
	if src == nil {
		return nil
	} else if txt, ok := src.(string); ok {
		*d = DeliveryStrategy(txt)
		return nil
	} else if txt, ok := src.([]byte); ok {
		*d = DeliveryStrategy(string(txt))
		return nil
	}
	return fmt.Errorf("Unsupported DeliveryStrategy: %#v", src)
}
