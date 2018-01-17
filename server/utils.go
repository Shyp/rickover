package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Shyp/go-types"
	"github.com/Shyp/rickover/models/queued_jobs"
	"github.com/Shyp/rickover/rest"
)

// getId validates that the provided ID is valid, and the prefix matches the
// expected prefix. Returns the correct ID, and a boolean describing whether
// the helper has written a response.
func getId(w http.ResponseWriter, r *http.Request, idStr string) (types.PrefixUUID, bool) {
	id, err := types.NewPrefixUUID(idStr)
	if err != nil {
		badRequest(w, r, &rest.Error{
			ID:    "invalid_uuid",
			Title: strings.Replace(err.Error(), "types: ", "", 1),
		})
		return id, true
	}
	if id.Prefix != queued_jobs.Prefix {
		badRequest(w, r, &rest.Error{
			ID:    "invalid_prefix",
			Title: fmt.Sprintf("Please use %s for the uuid prefix, not %s", queued_jobs.Prefix, id.Prefix),
		})
		return id, true
	}
	return id, false
}
