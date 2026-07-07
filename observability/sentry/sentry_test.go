package sentry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
)

func TestInitSentry_WhenEmptyDsn_ThenReturnsShutdown(t *testing.T) {
	cfg := &config.Config{
		App:    config.AppConfig{Environment: "test", Version: "1.0.0"},
		Sentry: config.SentryConfig{DSN: config.SecretURL(""), SampleRate: 1.0, FlushTimeout: time.Second},
	}

	shutdown, err := InitSentry(cfg, zap.NewNop())

	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitSentry_WhenValidConfig_ThenReturnsShutdownFunc(t *testing.T) {
	cfg := &config.Config{
		App:    config.AppConfig{Environment: "test", Version: "1.0.0"},
		Sentry: config.SentryConfig{DSN: config.SecretURL("https://key@sentry.io/123"), SampleRate: 1.0, FlushTimeout: time.Second},
	}

	shutdown, err := InitSentry(cfg, zap.NewNop())

	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitSentry_Shutdown_ThenReturnsNil(t *testing.T) {
	cfg := &config.Config{
		App:    config.AppConfig{Environment: "test", Version: "1.0.0"},
		Sentry: config.SentryConfig{DSN: config.SecretURL("https://key@sentry.io/123"), SampleRate: 0.5, FlushTimeout: time.Second},
	}

	shutdown, err := InitSentry(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	err = shutdown(context.Background())

	assert.NoError(t, err)
}
