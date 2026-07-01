package restapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	_ middleware.LogFormatter = (*LogFormatter)(nil)
	_ middleware.LogEntry     = (*logEntry)(nil)
)

// LogFormatter implements chi's LogFormatter interface to produce
// structured request logs via zap. If Skip returns true for a request,
// logging is suppressed for that request.
type LogFormatter struct {
	Logger *zap.Logger
	Skip   func(r *http.Request) bool
}

// NewLogEntry creates a new middleware.LogEntry for the given request.
func (l LogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	if l.Skip != nil && l.Skip(r) {
		return noopLogEntry{}
	}

	if l.Logger == nil {
		return noopLogEntry{}
	}

	ctx := r.Context()
	pattern := r.Pattern
	if pattern == "" {
		pattern = r.URL.Path
	}
	traceID := ""
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		traceID = sc.TraceID().String()
	}
	return &logEntry{
		reqID:     middleware.GetReqID(ctx),
		clientIP:  middleware.GetClientIP(ctx),
		host:      r.Host,
		method:    r.Method,
		path:      r.URL.Path,
		pattern:   pattern,
		userAgent: r.UserAgent(),
		traceID:   traceID,
		logger:    l.Logger,
	}
}

type logEntry struct {
	reqID     string
	clientIP  string
	host      string
	method    string
	path      string
	pattern   string
	userAgent string
	traceID   string
	logger    *zap.Logger
}

func (l logEntry) Write(status, bytes int, _ http.Header, elapsed time.Duration, extra any) {
	msg := fmt.Sprintf("%s %s", l.method, l.path)
	fields := []zap.Field{
		zap.Time("time", time.Now()),
		zap.Duration("latency", elapsed),
		zap.String("remote_ip", l.clientIP),
		zap.String("host", l.host),
		zap.String("method", l.method),
		zap.String("uri", l.path),
		zap.Int("status", status),
		zap.Int("size", bytes),
		zap.String("user_agent", l.userAgent),
		zap.String("trace_id", l.traceID),
	}
	if extra != nil {
		fields = append(fields, zap.Any("extra", extra))
	}
	switch {
	case status >= http.StatusInternalServerError:
		l.logger.Error(msg, fields...)
	case status >= http.StatusBadRequest:
		l.logger.Warn(msg, fields...)
	default:
		l.logger.Info(msg, fields...)
	}
}

func (l logEntry) Panic(v any, _ []byte) {
	middleware.PrintPrettyStack(v)
}

type noopLogEntry struct{}

func (l noopLogEntry) Write(_, _ int, _ http.Header, _ time.Duration, _ any) {
}

func (l noopLogEntry) Panic(_ any, _ []byte) {
}
