package middlewares

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newTracingRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

func TestNewOtelTracingMiddleware_WhenServiceNameEmpty_ThenNoOp(t *testing.T) {
	mw := NewOtelTracingMiddleware(OtelTracingMiddlewareConfig{})
	handler := mw(newHandler(http.StatusOK, "ok"))

	rec := httptest.NewRecorder()
	req := newTracingRequest(http.MethodGet, "/test")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestNewOtelTracingMiddleware_WhenSkipReturnsTrue_ThenPassesThrough(t *testing.T) {
	skipAll := func(r *http.Request) bool { return true }
	mw := NewOtelTracingMiddleware(OtelTracingMiddlewareConfig{
		ServiceName: "test-service",
		Skip:        skipAll,
	})
	handler := mw(newHandler(http.StatusOK, "ok"))

	rec := httptest.NewRecorder()
	req := newTracingRequest(http.MethodGet, "/skip")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestNewOtelTracingMiddleware_WhenRequestServed_ThenSpanCreated(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	mw := NewOtelTracingMiddleware(OtelTracingMiddlewareConfig{
		ServiceName: "test-service",
	})
	handler := mw(newHandler(http.StatusOK, "ok"))

	rec := httptest.NewRecorder()
	req := newTracingRequest(http.MethodGet, "/test-path")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	spans := sr.Ended()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "GET /test-path", span.Name())
	assert.Equal(t, "test-service", span.InstrumentationScope().Name)

	attrs := span.Attributes()

	var methodAttr, pathAttr, statusAttr string
	for _, a := range attrs {
		switch string(a.Key) {
		case "http.request.method":
			methodAttr = a.Value.AsString()
		case "url.path":
			pathAttr = a.Value.AsString()
		case "http.response.status_code":
			statusAttr = fmt.Sprintf("%d", a.Value.AsInt64())
		}
	}

	assert.Equal(t, "GET", methodAttr)
	assert.Equal(t, "/test-path", pathAttr)
	assert.Equal(t, "200", statusAttr)
}

func TestNewOtelTracingMiddleware_WhenErrorStatus_ThenSpanSetToError(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	mw := NewOtelTracingMiddleware(OtelTracingMiddlewareConfig{
		ServiceName: "test-service",
	})
	handler := mw(newHandler(http.StatusInternalServerError, "error"))

	rec := httptest.NewRecorder()
	req := newTracingRequest(http.MethodGet, "/error")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	spans := sr.Ended()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, codes.Error.String(), span.Status().Code.String())
}

func TestNewOtelTracingMiddleware_WhenSkipReturnsTrue_ThenNoSpanCreated(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	skipTest := func(r *http.Request) bool {
		return r.URL.Path == "/skip-me"
	}
	mw := NewOtelTracingMiddleware(OtelTracingMiddlewareConfig{
		ServiceName: "test-service",
		Skip:        skipTest,
	})
	handler := mw(newHandler(http.StatusOK, "ok"))

	rec := httptest.NewRecorder()
	req := newTracingRequest(http.MethodGet, "/skip-me")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, sr.Ended())
}

func TestNewOtelTracingMiddleware_WhenVariousMethods_ThenCorrectMethodAttribute(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	mw := NewOtelTracingMiddleware(OtelTracingMiddlewareConfig{
		ServiceName: "test-service",
	})

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		sr.Reset()

		handler := mw(newHandler(http.StatusOK, "ok"))
		rec := httptest.NewRecorder()
		req := newTracingRequest(method, "/resource")
		handler.ServeHTTP(rec, req)

		spans := sr.Ended()
		assert.Len(t, spans, 1)

		span := spans[0]
		assert.Equal(t, method+" /resource", span.Name())

		var foundMethod bool
		for _, a := range span.Attributes() {
			if string(a.Key) == "http.request.method" {
				assert.Equal(t, method, a.Value.AsString())
				foundMethod = true
			}
		}
		assert.True(t, foundMethod, "http.request.method attribute should be present")
	}
}
