package observability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
)

var globalTracerProvider *sdktrace.TracerProvider

// GetTelemetryProvider returns the global TracerProvider created by InitTelemetry.
// Returns nil if InitTelemetry has not been called.
func GetTelemetryProvider() *sdktrace.TracerProvider {
	return globalTracerProvider
}

// ResetTelemetryProvider clears the global TracerProvider. It is intended for
// use in tests that need to isolate global state between test cases.
func ResetTelemetryProvider() {
	globalTracerProvider = nil
}

func InitTelemetry(ctx context.Context, cfg *config.Config, logger *zap.Logger) (func(context.Context) error, error) {
	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}
	if !cfg.Telemetry.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	if cfg.Telemetry.SampleRate < 0 || cfg.Telemetry.SampleRate > 1.0 {
		return nil, fmt.Errorf("sample rate must be in [0.0, 1.0], got %f", cfg.Telemetry.SampleRate)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	res, err := resource.New(timeoutCtx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.App.Name),
			semconv.ServiceVersion(cfg.App.Version),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var tp *sdktrace.TracerProvider

	if cfg.Telemetry.TracingEndpoint == "" {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.Telemetry.SampleRate)),
		)
	} else {
		exporter, err := otlptracegrpc.New(timeoutCtx,
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(cfg.Telemetry.TracingEndpoint),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create exporter: %w", err)
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.Telemetry.SampleRate)),
		)
	}

	globalTracerProvider = tp
	otel.SetTracerProvider(tp)

	logger.Info("telemetry initialized",
		zap.String("endpoint", cfg.Telemetry.TracingEndpoint),
		zap.Float64("sample_rate", cfg.Telemetry.SampleRate),
	)

	return func(ctx context.Context) error {
		logger.Info("shutting down telemetry")
		err := tp.Shutdown(ctx)
		if err != nil {
			logger.Error("failed to shutdown telemetry", zap.Error(err))
		}
		return err
	}, nil
}
