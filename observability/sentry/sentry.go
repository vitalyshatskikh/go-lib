package sentry

import (
	"context"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
)

// InitSentry initializes the Sentry SDK with the configured DSN, environment,
// release version, and traces sample rate. It returns a shutdown function that
// flushes buffered events before the application exits.
func InitSentry(cfg *config.Config, logger *zap.Logger) (func(context.Context) error, error) {
	if cfg.Sentry.DSN.SecretValue() == "" {
		return func(context.Context) error { return nil }, nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.Sentry.DSN.SecretValue(),
		Environment:      cfg.App.Environment,
		Release:          cfg.App.Version,
		AttachStacktrace: true,
		// Set SampleRate to 1.0 to capture 100% of events.
		// We recommend adjusting this value in production.
		// Note: 0.0 is the same as 1.0, set empty Dsn to disable.
		SampleRate: cfg.Sentry.SampleRate,
		Debug:      cfg.Sentry.Debug,
	})
	if err != nil {
		return nil, err
	}
	logger.Info("sentry initialized",
		zap.String("endpoint", cfg.Sentry.DSN.String()),
		zap.Float64("sample_rate", cfg.Sentry.SampleRate),
	)
	return func(ctx context.Context) error {
		logger.Info("shutting down sentry")
		sentry.Flush(cfg.Sentry.FlushTimeout)
		return nil
	}, nil
}

// CaptureError reports an error to Sentry using the hub from the provided
// context. If no Sentry hub is found in the context, the error is silently
// dropped.
func CaptureError(ctx context.Context, err error) {
	if h := sentry.GetHubFromContext(ctx); h != nil {
		h.CaptureException(err)
	}
}
