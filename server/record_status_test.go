package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shyp/rest"
	"github.com/Shyp/rickover/test"
)

func TestNoBody400(t *testing.T) {
	t.Parallel()
	req, _ := http.NewRequest("POST", "/v1/jobs/echo/job_123", nil)
	req.SetBasicAuth("test", "password")
	w := httptest.NewRecorder()
	Get(u).ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusBadRequest)
	var err rest.Error
	e := json.Unmarshal(w.Body.Bytes(), &err)
	test.AssertNotError(t, e, "unmarshaling body")
	test.AssertEquals(t, err.ID, "missing_parameter")
}
