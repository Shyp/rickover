// A simple http.Handler that can match wildcard routes, and call the
// appropriate handler.
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type route struct {
	pattern *regexp.Regexp
	methods []string
	handler http.Handler
}

func buildRoute(regex string) *regexp.Regexp {
	route, err := regexp.Compile(regex)
	if err != nil {
		log.Fatal(err)
	}
	return route
}

type RegexpHandler struct {
	routes []*route
}

func (h *RegexpHandler) Handler(pattern *regexp.Regexp, methods []string, handler http.Handler) {
	h.routes = append(h.routes, &route{
		pattern: pattern,
		methods: methods,
		handler: handler,
	})
}

func (h *RegexpHandler) HandleFunc(pattern *regexp.Regexp, methods []string, handler func(http.ResponseWriter, *http.Request)) {
	h.routes = append(h.routes, &route{
		pattern: pattern,
		methods: methods,
		handler: http.HandlerFunc(handler),
	})
}

func (h *RegexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if route.pattern.MatchString(r.URL.Path) {
			upperMethod := strings.ToUpper(r.Method)
			for _, method := range route.methods {
				if strings.ToUpper(method) == upperMethod {
					route.handler.ServeHTTP(w, r)
					return
				}
			}
			if upperMethod == "OPTIONS" {
				methods := strings.Join(append(route.methods, "OPTIONS"), ", ")
				w.Header().Set("Allow", methods)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
				json.NewEncoder(w).Encode(new405(r))
			}
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(new404(r))
}
