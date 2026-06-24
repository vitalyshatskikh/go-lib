package middlewares

import (
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/prometheus/client_golang/prometheus"
)

func resetPrometheusRegistry() {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
}

func newGetRequest(path string) *http.Request {
	return httptest.NewRequest(http.MethodGet, path, nil)
}

func newHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}
