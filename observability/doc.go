// Package observability provides factory functions to initialize the application's
// observability subsystems: structured logging (zap), Prometheus metrics, and
// OpenTelemetry tracing.
//
// Each Init* function returns a shutdown function (func(context.Context) error)
// suitable for use with the closer package. Call Init* functions exactly once
// at application startup; repeated calls may leak resources.
package observability
