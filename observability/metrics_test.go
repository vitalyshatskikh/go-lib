package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
)

func TestInitMetrics_WhenDisabled_ThenReturnsNoopShutdown(t *testing.T) {
	cfg := &config.Config{
		Metrics: config.MetricsConfig{Enabled: false},
	}
	logger := zap.NewNop()

	shutdown, err := InitMetrics(cfg, logger)

	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitMetrics_WhenNilLogger_ThenReturnsError(t *testing.T) {
	cfg := &config.Config{
		Metrics: config.MetricsConfig{Enabled: true},
	}

	shutdown, err := InitMetrics(cfg, nil)

	require.Error(t, err)
	assert.Nil(t, shutdown)
	assert.Contains(t, err.Error(), "logger must not be nil")
}

func TestInitMetrics_WhenEnabled_ThenReturnsShutdownFunc(t *testing.T) {
	cfg := &config.Config{
		Metrics: config.MetricsConfig{
			Enabled: true,
			Host:    "127.0.0.1",
			Port:    0,
			Path:    "/metrics",
		},
	}
	logger := zap.NewNop()

	shutdown, err := InitMetrics(cfg, logger)

	require.NoError(t, err)
	require.NotNil(t, shutdown)
}

func TestInitMetrics_WhenEmptyPath_ThenDoesNotPanic(t *testing.T) {
	cfg := &config.Config{
		Metrics: config.MetricsConfig{
			Enabled: true,
			Host:    "127.0.0.1",
			Port:    0,
			Path:    "",
		},
	}
	logger := zap.NewNop()

	shutdown, err := InitMetrics(cfg, logger)

	require.NoError(t, err)
	require.NotNil(t, shutdown)
}
