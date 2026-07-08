package sentry

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
	sentryotlp "github.com/getsentry/sentry-go/otel/otlp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
	"github.com/vitalyshatskikh/go-lib/observability"
)

// InitSentry initializes the Sentry SDK with the configured DSN, environment,
// release version, and traces sample rate. If Sentry.EnableTracing is true,
// it also registers the Sentry OTLP span exporter on the global TracerProvider
// and adds the OTel context-linking integration. It returns a shutdown function
// that flushes buffered events before the application exits.
func InitSentry(ctx context.Context, cfg *config.Config, logger *zap.Logger) (func(context.Context) error, error) {
	if cfg.Sentry.DSN.SecretValue() == "" {
		return func(context.Context) error { return nil }, nil
	}

	integrations := func(integrations []sentry.Integration) []sentry.Integration {
		return integrations
	}

	if cfg.Sentry.EnableTracing {
		tp := observability.GetTelemetryProvider()
		if tp == nil {
			return nil, fmt.Errorf("telemetry must be initialized before sentry when EnableTracing is true")
		}

		exporter, err := sentryotlp.NewTraceExporter(ctx, cfg.Sentry.DSN.SecretValue())
		if err != nil {
			return nil, fmt.Errorf("failed to create sentry otlp exporter: %w", err)
		}

		tp.RegisterSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter))

		integrations = func(integrations []sentry.Integration) []sentry.Integration {
			return append(integrations, sentryotel.NewOtelIntegration())
		}
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.Sentry.DSN.SecretValue(),
		Environment:      cfg.App.Environment,
		Release:          cfg.App.Version,
		AttachStacktrace: true,
		SampleRate:       cfg.Sentry.SampleRate,
		Debug:            cfg.Sentry.Debug,
		Integrations:     integrations,
	})
	if err != nil {
		return nil, err
	}

	logger.Info("sentry initialized",
		zap.String("endpoint", cfg.Sentry.DSN.String()),
		zap.Float64("sample_rate", cfg.Sentry.SampleRate),
		zap.Bool("tracing_enabled", cfg.Sentry.EnableTracing),
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
