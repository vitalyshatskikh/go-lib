package middlewares

import (
	"io"
	"net/http"
	"net/http/httptest"
)

func resetMetricValues() {
	requestCount.Reset()
	requestDuration.Reset()
	responseSize.Reset()
	requestSize.Reset()
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
