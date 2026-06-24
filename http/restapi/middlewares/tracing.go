package middlewares

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// OtelTracingMiddlewareConfig configures the OpenTelemetry tracing middleware.
type OtelTracingMiddlewareConfig struct {
	// ServiceName is used as the tracer name for creating spans.
	// If empty, the middleware is a no-op.
	ServiceName string

	// Skip optionally specifies a predicate that determines which requests
	// should not be traced.
	Skip func(r *http.Request) bool
}

// NewOtelTracingMiddleware creates a net/http compatible middleware that
// extracts existing tracing context, creates a new server span, records
// HTTP semantic attributes, and injects the tracing context into response
// headers. When Skip returns true, no tracing occurs and the request is
// passed through unchanged.
func NewOtelTracingMiddleware(conf OtelTracingMiddlewareConfig) func(next http.Handler) http.Handler {
	if conf.ServiceName == "" {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if conf.Skip != nil && conf.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			ctx := otel.GetTextMapPropagator().Extract(
				r.Context(),
				propagation.HeaderCarrier(r.Header),
			)

			pattern := r.Pattern
			if pattern == "" {
				pattern = r.URL.Path
			}

			spanName := fmt.Sprintf("%s %s", r.Method, pattern)
			ctx, span := otel.Tracer(conf.ServiceName).Start(
				ctx,
				spanName,
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			span.SetAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.URLPathKey.String(r.URL.Path),
				semconv.HTTPRouteKey.String(pattern),
			)

			r = r.WithContext(ctx)

			otel.GetTextMapPropagator().Inject(
				ctx,
				propagation.HeaderCarrier(w.Header()),
			)

			ww, ok := w.(middleware.WrapResponseWriter)
			if !ok {
				ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			}

			next.ServeHTTP(ww, r)

			statusCode := ww.Status()
			span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(statusCode))
			if statusCode < 500 {
				span.SetStatus(codes.Ok, "")
			} else {
				span.SetStatus(codes.Error, fmt.Sprintf("received %d HTTP status code", statusCode))
			}
		}

		return http.HandlerFunc(fn)
	}
}
