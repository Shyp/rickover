package test_jobs

import (
	"fmt"
	"testing"
	"time"

	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/test"
)

func TestCreateMissingFields(t *testing.T) {
	t.Parallel()
	test.SetUp(t)
	job := models.Job{
		Name: "email-signup",
	}
	_, err := jobs.Create(job)
	test.AssertError(t, err, "")
	test.AssertEquals(t, err.Error(), "Invalid delivery_strategy: \"\"")
}

func TestCreateInvalidFields(t *testing.T) {
	t.Parallel()
	test.SetUp(t)
	job := models.Job{
		Name:             "email-signup",
		DeliveryStrategy: models.DeliveryStrategy("foo"),
	}
	_, err := jobs.Create(job)
	test.AssertError(t, err, "")
	test.AssertEquals(t, err.Error(), "Invalid delivery_strategy: \"foo\"")
}

var sampleJob = models.Job{
	Name:             "email-signup",
	DeliveryStrategy: models.StrategyAtLeastOnce,
	Attempts:         3,
	Concurrency:      1,
}

func TestCreateReturnsRecord(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	j, err := jobs.Create(sampleJob)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, j.Name, "email-signup")
	test.AssertEquals(t, j.DeliveryStrategy, models.StrategyAtLeastOnce)
	test.AssertEquals(t, j.Attempts, uint8(3))
	test.AssertEquals(t, j.Concurrency, uint8(1))
	diff := time.Since(j.CreatedAt)
	test.Assert(t, diff < 100*time.Millisecond, fmt.Sprintf("CreatedAt should be close to the current time, got %v", diff))
}

func TestGet(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	_, err := jobs.Create(sampleJob)
	test.AssertNotError(t, err, "")
	j, err := jobs.Get("email-signup")
	test.AssertEquals(t, j.Name, "email-signup")
	test.AssertEquals(t, j.DeliveryStrategy, models.StrategyAtLeastOnce)
	test.AssertEquals(t, j.Attempts, uint8(3))
	test.AssertEquals(t, j.Concurrency, uint8(1))
	diff := time.Since(j.CreatedAt)
	test.Assert(t, diff < 20*time.Millisecond, "")
}
