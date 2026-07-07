// Package http provides Sentry HTTP middleware that wraps an http.Handler
// with Sentry's panic recovery and error reporting.
//
// The WrapHandler function wraps an http.Handler with Sentry middleware
// that injects a Sentry hub into the request context and recovers panics.
package http
