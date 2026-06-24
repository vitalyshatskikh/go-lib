package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
)

func TestInitTelemetry_WhenDisabled_ThenReturnsNoopShutdown(t *testing.T) {
	cfg := &config.Config{
		Telemetry: config.TelemetryConfig{Enabled: false},
	}
	logger := zap.NewNop()

	shutdown, err := InitTelemetry(context.Background(), cfg, logger)

	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitTelemetry_WhenNilLogger_ThenReturnsError(t *testing.T) {
	cfg := &config.Config{
		Telemetry: config.TelemetryConfig{Enabled: true},
	}

	shutdown, err := InitTelemetry(context.Background(), cfg, nil)

	require.Error(t, err)
	assert.Nil(t, shutdown)
	assert.Contains(t, err.Error(), "logger must not be nil")
}

func TestInitTelemetry_WhenInvalidSampleRate_ThenReturnsError(t *testing.T) {
	cfg := &config.Config{
		Telemetry: config.TelemetryConfig{
			Enabled:    true,
			SampleRate: -0.5,
		},
	}
	logger := zap.NewNop()

	shutdown, err := InitTelemetry(context.Background(), cfg, logger)

	require.Error(t, err)
	assert.Nil(t, shutdown)
	assert.Contains(t, err.Error(), "sample rate must be in [0.0, 1.0]")
}

func TestInitTelemetry_WhenSampleRateAbove1_ThenReturnsError(t *testing.T) {
	cfg := &config.Config{
		Telemetry: config.TelemetryConfig{
			Enabled:    true,
			SampleRate: 1.5,
		},
	}
	logger := zap.NewNop()

	shutdown, err := InitTelemetry(context.Background(), cfg, logger)

	require.Error(t, err)
	assert.Nil(t, shutdown)
	assert.Contains(t, err.Error(), "sample rate must be in [0.0, 1.0]")
}
