// Package zap provides a zap.Logger wrapper that forwards log entries
// (ErrorLevel by default) to Sentry via a custom zapcore.Core.
//
// The WrapLogger function wraps a *zap.Logger with a Sentry core, enabling
// automatic error reporting from structured logs.
package zap
