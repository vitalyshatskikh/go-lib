package observability

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/vitalyshatskikh/go-lib/config"
)

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

	timeoutCtx, cancel := context.WithTimeout(ctx, cfg.Telemetry.ConnectTimeout)
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
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Telemetry.TracingEndpoint)}

		if cfg.Telemetry.UseTLS {
			tlsConfig, err := buildTLSConfig(&cfg.Telemetry)
			if err != nil {
				return nil, fmt.Errorf("failed to build TLS config: %w", err)
			}
			opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))))
		} else {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}

		exporter, err := otlptracegrpc.New(timeoutCtx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create exporter: %w", err)
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.Telemetry.SampleRate)),
		)
	}

	otel.SetTracerProvider(tp)

	logger.Info("telemetry initialized",
		zap.String("endpoint", cfg.Telemetry.TracingEndpoint),
		zap.Float64("sample_rate", cfg.Telemetry.SampleRate),
		zap.Bool("use_tls", cfg.Telemetry.UseTLS),
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

func buildTLSConfig(cfg *config.TelemetryConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.SkipVerifyCA,
	}

	if cfg.CACertPath != "" {
		caCert, err := os.ReadFile(cfg.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to parse CA cert")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}
