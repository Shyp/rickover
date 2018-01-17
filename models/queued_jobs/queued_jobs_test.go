package queued_jobs

import (
	"fmt"

	"github.com/Shyp/go-types"
)

func ExampleDelete() {
	id, _ := types.NewPrefixUUID("job_6740b44e-13b9-475d-af06-979627e0e0d6")
	err := Delete(id)
	// Returns an error, because we didn't insert rows into the database.
	fmt.Println(err)
}
