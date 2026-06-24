package middlewares

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	_           = iota // ignore first value by assigning to blank identifier
	bKB float64 = 1 << (10 * iota)
	bMB
)

var sizeBuckets = []float64{1.0 * bKB, 2.0 * bKB, 5.0 * bKB, 10.0 * bKB, 100 * bKB, 500 * bKB, 1.0 * bMB, 2.5 * bMB, 5.0 * bMB, 10.0 * bMB}

// PrometheusMiddlewareConfig configures the Prometheus HTTP metrics middleware.
type PrometheusMiddlewareConfig struct {
	Skip func(r *http.Request) bool
}

// NewPrometheusMiddleware returns net/http compatible middleware that records HTTP request
// count, duration, request size and response size as Prometheus histograms
// and counters, partitioned by status code, method, host, and path.
func NewPrometheusMiddleware(conf PrometheusMiddlewareConfig) func(next http.Handler) http.Handler {
	registerer := prometheus.DefaultRegisterer

	labelNames := []string{"status_code", "method", "host", "path"}
	requestCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "",
			Subsystem: "http_server",
			Name:      "requests_total",
			Help:      "How many HTTP requests processed, partitioned by status code and HTTP method.",
		},
		labelNames,
	)
	registerer.MustRegister(requestCount)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "",
			Subsystem: "http_server",
			Name:      "request_duration_seconds",
			Help:      "The HTTP request latencies in seconds.",
			// Here, we use the prometheus defaults which are for ~10s request length max:
			// []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
			Buckets: prometheus.DefBuckets,
		},
		labelNames,
	)
	registerer.MustRegister(requestDuration)

	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "",
			Subsystem: "http_server",
			Name:      "response_size_bytes",
			Help:      "The HTTP response sizes in bytes.",
			Buckets:   sizeBuckets,
		},
		labelNames,
	)
	registerer.MustRegister(responseSize)

	requestSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "",
			Subsystem: "http_server",
			Name:      "request_size_bytes",
			Help:      "The HTTP request sizes in bytes.",
			Buckets:   sizeBuckets,
		},
		labelNames,
	)
	registerer.MustRegister(requestSize)

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if conf.Skip != nil && conf.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Because of net/http handler signature that returns nothing
			// we need wrap response to intercept status/size/etc.
			// Maybe already wrapped by previous middleware (Logger)
			ww, ok := w.(middleware.WrapResponseWriter)
			if !ok {
				ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			}

			reqSz := computeApproximateRequestSize(r)

			start := time.Now()
			next.ServeHTTP(ww, r)
			elapsed := float64(time.Since(start)) / float64(time.Second)

			pattern := r.Pattern
			if pattern == "" {
				pattern = r.URL.Path
			}

			status := ww.Status()
			respSz := ww.BytesWritten()

			values := []string{
				strconv.Itoa(status),
				r.Method,
				r.Host,
				pattern,
			}

			requestDuration.WithLabelValues(values...).Observe(elapsed)
			requestCount.WithLabelValues(values...).Inc()
			requestSize.WithLabelValues(values...).Observe(float64(reqSz))
			responseSize.WithLabelValues(values...).Observe(float64(respSz))
		}

		return http.HandlerFunc(fn)
	}
}

func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}
