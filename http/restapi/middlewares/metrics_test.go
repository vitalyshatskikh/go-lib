package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeApproximateRequestSize_WhenEmptyRequest_ThenReturnsZero(t *testing.T) {
	r := &http.Request{}
	size := computeApproximateRequestSize(r)
	assert.Equal(t, 0, size)
}

func TestComputeApproximateRequestSize_WhenRequestHasURL_ThenIncludesPathLen(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/hello/world", nil)
	r.Proto = "HTTP/1.1"

	size := computeApproximateRequestSize(r)

	assert.Equal(t, len(r.URL.Path)+len(r.Method)+len(r.Proto)+len(r.Host), size)
}

func TestComputeApproximateRequestSize_WhenRequestHasHeaders_ThenIncludesHeaderSizes(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set("X-Custom", "value1")
	r.Header.Set("Authorization", "Bearer token123")

	size := computeApproximateRequestSize(r)
	expectedHeaderSize := len("X-Custom") + len("value1") + len("Authorization") + len("Bearer token123")

	assert.GreaterOrEqual(t, size, expectedHeaderSize)
}

func TestComputeApproximateRequestSize_WhenRequestHasContentLength_ThenIncludesBodySize(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/upload", nil)
	r.ContentLength = 1024

	size := computeApproximateRequestSize(r)

	assert.Equal(t, len(r.URL.Path)+len(r.Method)+len(r.Proto)+len(r.Host)+int(r.ContentLength), size)
}

func TestComputeApproximateRequestSize_WhenNegativeContentLength_ThenIgnoresBodySize(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/path", nil)
	r.ContentLength = -1

	expected := len(r.URL.Path) + len(r.Method) + len(r.Proto) + len(r.Host)
	size := computeApproximateRequestSize(r)

	assert.Equal(t, expected, size)
}

func TestComputeApproximateRequestSize_WhenNilURL_ThenDoesNotPanic(t *testing.T) {
	r := &http.Request{
		Method: http.MethodGet,
		Proto:  "HTTP/1.1",
		Host:   "example.com",
	}

	size := computeApproximateRequestSize(r)

	expected := len(r.Method) + len(r.Proto) + len(r.Host)
	assert.Equal(t, expected, size)
}

func TestComputeApproximateRequestSize_WhenMultipleHeaderValues_ThenCountsAll(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/path", nil)
	r.Header.Add("X-Custom", "val1")
	r.Header.Add("X-Custom", "val2")

	size := computeApproximateRequestSize(r)
	// header name counted once, both values counted
	expectedHeaderPart := len("X-Custom") + len("val1") + len("val2")

	assert.GreaterOrEqual(t, size, expectedHeaderPart)
}

func TestNewPrometheusMiddleware_WhenRequestServed_ThenRecordsAllMetrics(t *testing.T) {
	resetPrometheusRegistry()

	mw := NewPrometheusMiddleware(PrometheusMiddlewareConfig{})
	handler := mw(newHandler(http.StatusOK, "ok"))

	rec := httptest.NewRecorder()
	req := newGetRequest("/test")
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Gather and verify metrics via the test registry
	gatherer := prometheus.DefaultRegisterer.(*prometheus.Registry)
	families, err := gatherer.Gather()
	require.NoError(t, err)

	familyNames := make(map[string]bool)
	for _, f := range families {
		familyNames[f.GetName()] = true
	}

	assert.True(t, familyNames["http_server_requests_total"], "requests_total metric should exist")
	assert.True(t, familyNames["http_server_request_duration_seconds"], "request_duration_seconds metric should exist")
	assert.True(t, familyNames["http_server_response_size_bytes"], "response_size_bytes metric should exist")
	assert.True(t, familyNames["http_server_request_size_bytes"], "request_size_bytes metric should exist")

	// Verify the request counter was incremented
	for _, f := range families {
		if f.GetName() == "http_server_requests_total" {
			require.Len(t, f.GetMetric(), 1)
			assert.InEpsilon(t, float64(1), f.GetMetric()[0].GetCounter().GetValue(), float64(0))
			// Verify labels
			labels := f.GetMetric()[0].GetLabel()
			labelMap := make(map[string]string)
			for _, l := range labels {
				labelMap[l.GetName()] = l.GetValue()
			}
			assert.Equal(t, "200", labelMap["status_code"])
			assert.Equal(t, http.MethodGet, labelMap["method"])
			assert.Equal(t, "/test", labelMap["path"])
			assert.Equal(t, "example.com", labelMap["host"])
		}
	}
}

func TestNewPrometheusMiddleware_WhenSkipReturnsTrue_ThenSkipsMetrics(t *testing.T) {
	resetPrometheusRegistry()

	skipAll := func(r *http.Request) bool { return true }
	mw := NewPrometheusMiddleware(PrometheusMiddlewareConfig{Skip: skipAll})
	handler := mw(newHandler(http.StatusOK, "ok"))

	rec := httptest.NewRecorder()
	req := newGetRequest("/skip")
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Metrics should NOT be recorded — this is the bug location
	gatherer := prometheus.DefaultRegisterer.(*prometheus.Registry)
	families, err := gatherer.Gather()
	require.NoError(t, err)

	for _, f := range families {
		if f.GetName() == "http_server_requests_total" {
			// If we get here, metrics were recorded despite Skip returning true
			t.Errorf("expected no metrics when Skip returns true, but %s has %d metrics",
				f.GetName(), len(f.GetMetric()))
		}
	}
}

func TestNewPrometheusMiddleware_WhenResponseWriterAlreadyWrapped_ThenReusesIt(t *testing.T) {
	resetPrometheusRegistry()

	mw := NewPrometheusMiddleware(PrometheusMiddlewareConfig{})

	var capturedWriter middleware.WrapResponseWriter
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		capturedWriter, ok = w.(middleware.WrapResponseWriter)
		if !ok {
			t.Error("expected wrapped response writer")
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := mw(innerHandler)

	rec := httptest.NewRecorder()
	req := newGetRequest("/wrapped")
	preWrapped := middleware.NewWrapResponseWriter(rec, req.ProtoMajor)
	handler.ServeHTTP(preWrapped, req)

	// The middleware should reuse the same wrapper, not create a new one
	assert.Equal(
		t,
		rec,
		capturedWriter.Unwrap(),
		"should reuse the existing WrapResponseWriter",
	)
}

func TestNewPrometheusMiddleware_WhenMultipleStatusCodes_ThenLabelsAreCorrect(t *testing.T) {
	resetPrometheusRegistry()

	mw := NewPrometheusMiddleware(PrometheusMiddlewareConfig{})
	router := http.NewServeMux()
	router.HandleFunc("/ok", newHandler(http.StatusOK, "ok"))
	router.HandleFunc("/notfound", newHandler(http.StatusNotFound, "nope"))
	handler := mw(router)

	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/ok", http.StatusOK},
		{"/notfound", http.StatusNotFound},
	} {
		rec := httptest.NewRecorder()
		req := newGetRequest(tc.path)
		handler.ServeHTTP(rec, req)
		assert.Equal(t, tc.status, rec.Code)
	}

	gatherer := prometheus.DefaultRegisterer.(*prometheus.Registry)
	families, err := gatherer.Gather()
	require.NoError(t, err)

	for _, f := range families {
		if f.GetName() == "http_server_requests_total" {
			require.Len(t, f.GetMetric(), 2, "should have two distinct label combinations")

			statuses := make(map[string]bool)
			for _, m := range f.GetMetric() {
				for _, l := range m.GetLabel() {
					if l.GetName() == "status_code" {
						statuses[l.GetValue()] = true
					}
				}
			}
			assert.True(t, statuses["200"], "should have 200 status code label")
			assert.True(t, statuses["404"], "should have 404 status code label")
		}
	}
}

func TestNewPrometheusMiddleware_WhenPathPatternEmpty_ThenFallsBackToURLPath(t *testing.T) {
	resetPrometheusRegistry()

	mw := NewPrometheusMiddleware(PrometheusMiddlewareConfig{})
	handler := mw(newHandler(http.StatusOK, "ok"))

	rec := httptest.NewRecorder()
	req := newGetRequest("/some/path")
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	gatherer := prometheus.DefaultRegisterer.(*prometheus.Registry)
	families, err := gatherer.Gather()
	require.NoError(t, err)

	for _, f := range families {
		if f.GetName() == "http_server_requests_total" {
			require.Len(t, f.GetMetric(), 1)
			for _, l := range f.GetMetric()[0].GetLabel() {
				if l.GetName() == "path" {
					assert.Equal(t, "/some/path", l.GetValue(), "should fall back to URL path when Pattern is empty")
				}
			}
		}
	}
}
