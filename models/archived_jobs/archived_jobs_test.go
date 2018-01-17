package archived_jobs

import (
	"fmt"

	"github.com/Shyp/go-types"
	"github.com/Shyp/rickover/models"
)

func ExampleCreate() {
	id, _ := types.NewPrefixUUID("job_6740b44e-13b9-475d-af06-979627e0e0d6")
	aj, _ := Create(id, "echo", models.StatusSucceeded, 3)
	fmt.Println(aj.ID.String())
}

func ExampleGet() {
	id, _ := types.NewPrefixUUID("job_6740b44e-13b9-475d-af06-979627e0e0d6")
	aj, _ := Get(id)
	fmt.Println(aj.ID.String())
}
