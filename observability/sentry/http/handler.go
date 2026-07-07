package http

import (
	"net/http"

	sentryhttp "github.com/getsentry/sentry-go/http"

	"github.com/vitalyshatskikh/go-lib/config"
)

// WrapHandler wraps the provided http.Handler with Sentry HTTP middleware that
// captures panics and errors reported via sentry-go. If cfg.Sentry.DSN is empty,
// returns the original handler unchanged.
func WrapHandler(cfg *config.Config, h http.Handler) http.Handler {
	if cfg.Sentry.DSN.SecretValue() == "" {
		return h
	}

	opts := sentryhttp.Options{
		Repanic: true,
	}
	return sentryhttp.New(opts).Handle(h)
}
