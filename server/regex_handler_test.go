package server

import (
	"net/http"
	"regexp"
)

func ExampleRegexpHandler() {
	// GET /v1/jobs/:job-name
	route := regexp.MustCompile(`^/v1/jobs/(?P<JobName>[^\s\/]+)$`)

	h := new(RegexpHandler)
	h.HandleFunc(route, []string{"GET", "POST"}, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})
}
