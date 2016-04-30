package downstream

import (
	"encoding/json"

	"github.com/Shyp/rickover/Godeps/_workspace/src/github.com/Shyp/go-types"
)

func ExampleClient() {
	client := NewClient("test", "hymanrickover", "http://downstream-server.example.com")
	params := JobParams{
		Data:     json.RawMessage([]byte("{}")),
		Attempts: 3,
	}
	id, _ := types.NewPrefixUUID("job_123")
	client.Job.Post("invoice-shipment", &id, &params)
}
