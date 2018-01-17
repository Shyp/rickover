package downstream

import (
	"encoding/json"

	"github.com/Shyp/go-types"
)

// Create a new client, then make a request to a downstream service with an
// empty data object.
func Example() {
	client := NewClient("test", "hymanrickover", "http://downstream-server.example.com")
	params := JobParams{
		Data:     json.RawMessage([]byte("{}")),
		Attempts: 3,
	}
	id, _ := types.NewPrefixUUID("job_123")
	client.Job.Post("invoice-shipment", &id, &params)
}
