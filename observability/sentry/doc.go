// Package sentry provides Sentry error tracking initialization.
//
// Sub-packages:
//   - http — HTTP middleware that wraps an http.Handler with Sentry panic recovery
//   - zap  — zap.Logger wrapper that forwards log entries to Sentry
//   - mock — test helper for running a local Sentry mock server
package sentry
