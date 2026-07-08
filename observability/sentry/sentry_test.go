package sentry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
	"github.com/vitalyshatskikh/go-lib/observability"
)

func TestInitSentry_WhenEmptyDsn_ThenReturnsShutdown(t *testing.T) {
	cfg := &config.Config{
		App:    config.AppConfig{Environment: "test", Version: "1.0.0"},
		Sentry: config.SentryConfig{DSN: config.SecretURL(""), SampleRate: 1.0, FlushTimeout: 100 * time.Millisecond},
	}

	shutdown, err := InitSentry(context.Background(), cfg, zap.NewNop())

	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitSentry_WhenValidConfig_ThenReturnsShutdownFunc(t *testing.T) {
	cfg := &config.Config{
		App:    config.AppConfig{Environment: "test", Version: "1.0.0"},
		Sentry: config.SentryConfig{DSN: config.SecretURL("https://key@sentry.io/123"), SampleRate: 1.0, FlushTimeout: 100 * time.Millisecond},
	}

	shutdown, err := InitSentry(context.Background(), cfg, zap.NewNop())

	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitSentry_Shutdown_ThenReturnsNil(t *testing.T) {
	cfg := &config.Config{
		App:    config.AppConfig{Environment: "test", Version: "1.0.0"},
		Sentry: config.SentryConfig{DSN: config.SecretURL("https://key@sentry.io/123"), SampleRate: 0.5, FlushTimeout: 100 * time.Millisecond},
	}

	shutdown, err := InitSentry(context.Background(), cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	err = shutdown(context.Background())

	assert.NoError(t, err)
}

func TestInitSentry_WhenEnableTracingWithoutTp_ThenReturnsError(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{Environment: "test", Version: "1.0.0"},
		Sentry: config.SentryConfig{
			DSN:           config.SecretURL("https://key@sentry.io/123"),
			SampleRate:    1.0,
			FlushTimeout:  100 * time.Millisecond,
			EnableTracing: true,
		},
	}

	shutdown, err := InitSentry(context.Background(), cfg, zap.NewNop())

	require.Error(t, err)
	assert.Nil(t, shutdown)
}

func TestInitSentry_WhenEnableTracingWithTp_ThenReturnsShutdown(t *testing.T) {
	t.Cleanup(observability.ResetTelemetryProvider)

	_, err := observability.InitTelemetry(context.Background(), &config.Config{
		App: config.AppConfig{Name: "test", Version: "1.0.0"},
		Telemetry: config.TelemetryConfig{
			Enabled:    true,
			SampleRate: 1.0,
		},
	}, zap.NewNop())
	require.NoError(t, err)

	cfg := &config.Config{
		App: config.AppConfig{Environment: "test", Version: "1.0.0"},
		Sentry: config.SentryConfig{
			DSN:           config.SecretURL("https://key@sentry.io/123"),
			SampleRate:    1.0,
			FlushTimeout:  100 * time.Millisecond,
			EnableTracing: true,
		},
	}

	shutdown, err := InitSentry(context.Background(), cfg, zap.NewNop())

	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}
