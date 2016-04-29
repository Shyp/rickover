package factory

import (
	"testing"

	"github.com/Shyp/rickover/test"
)

func ExampleRandomId() {
	RandomId("job_")
	// Will return something like "job_5555b44e-13b9-475d-af06-979627e0e0d6"
}

func ExampleCreateQueuedJob() {
	t := &testing.T{}
	CreateQueuedJob(t, EmptyData)
}

func TestRandomId(t *testing.T) {
	id := RandomId("job_")
	test.AssertEquals(t, id.Prefix, "job_")
	test.AssertContains(t, id.String(), "job_")
	test.AssertNotNil(t, id.UUID, "")
}
