package downstream

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Shyp/go-types"
)

type JobService struct {
	Client *Client
}

type JobParams struct {
	Data     json.RawMessage `json:"data"`
	Attempts uint8           `json:"attempts"`
}

// Post makes a request to /v1/jobs/:job-name/:job-id with the job data.
// The downstream service is expected to respond with a 202, so there is no
// positive return value, only nil if the response was a 2xx status code.
func (j *JobService) Post(name string, id *types.PrefixUUID, jp *JobParams) error {
	if jp == nil || id == nil {
		return errors.New("no job to post")
	}
	if len(jp.Data) == 0 {
		jp.Data = []byte("null")
	}
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(jp)
	if err != nil {
		return err
	}
	req, err := j.Client.NewRequest("POST", fmt.Sprintf("/v1/jobs/%s/%s", name, id.String()), b)
	if err != nil {
		return err
	}
	var d struct{}
	return j.Client.Do(req, &d)
}
