package test_jobs

import (
	"fmt"
	"testing"
	"time"

	types "github.com/Shyp/go-types"
	"github.com/Shyp/rickover/models"
	"github.com/Shyp/rickover/models/jobs"
	"github.com/Shyp/rickover/test"
)

func TestAll(t *testing.T) {
	test.SetUp(t)
	defer test.TearDown(t)
	t.Run("Parallel", func(t *testing.T) {
		t.Run("CreateMissingFields", testCreateMissingFields)
		t.Run("CreateInvalidFields", testCreateInvalidFields)
		t.Run("CreateReturnsRecord", testCreateReturnsRecord)
		t.Run("Get", testGet)
	})
}

func testCreateMissingFields(t *testing.T) {
	t.Parallel()
	job := models.Job{
		Name: "email-signup",
	}
	_, err := jobs.Create(job)
	test.AssertError(t, err, "")
	test.AssertEquals(t, err.Error(), "Invalid delivery_strategy: \"\"")
}

func testCreateInvalidFields(t *testing.T) {
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

func newJob(t *testing.T) models.Job {
	t.Helper()
	id, _ := types.GenerateUUID("jobname_")
	return models.Job{
		Name:             id.String(),
		DeliveryStrategy: models.StrategyAtLeastOnce,
		Attempts:         3,
		Concurrency:      1,
	}
}

func testCreateReturnsRecord(t *testing.T) {
	t.Parallel()
	j0 := newJob(t)
	j, err := jobs.Create(j0)
	test.AssertNotError(t, err, "")
	test.AssertEquals(t, j.Name, j0.Name)
	test.AssertEquals(t, j.DeliveryStrategy, models.StrategyAtLeastOnce)
	test.AssertEquals(t, j.Attempts, uint8(3))
	test.AssertEquals(t, j.Concurrency, uint8(1))
	diff := time.Since(j.CreatedAt)
	test.Assert(t, diff < 100*time.Millisecond, fmt.Sprintf("CreatedAt should be close to the current time, got %v", diff))
}

func testGet(t *testing.T) {
	t.Parallel()
	j0 := newJob(t)
	_, err := jobs.Create(j0)
	test.AssertNotError(t, err, "")
	j, err := jobs.Get(j0.Name)
	test.AssertEquals(t, j.Name, j0.Name)
	test.AssertEquals(t, j.DeliveryStrategy, models.StrategyAtLeastOnce)
	test.AssertEquals(t, j.Attempts, uint8(3))
	test.AssertEquals(t, j.Concurrency, uint8(1))
	diff := time.Since(j.CreatedAt)
	test.Assert(t, diff < 100*time.Millisecond, "")
}
