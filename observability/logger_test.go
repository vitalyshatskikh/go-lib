package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vitalyshatskikh/go-lib/config"
)

func TestInitLogger_WhenValidLevel_ThenReturnsLogger(t *testing.T) {
	cfg := &config.Config{
		Logging: config.LoggingConfig{Level: "debug"},
	}

	logger, err := InitLogger(cfg)

	require.NoError(t, err)
	require.NotNil(t, logger)
	logger.Sync() //nolint:errcheck
}

func TestInitLogger_WhenInvalidLevel_ThenReturnsError(t *testing.T) {
	cfg := &config.Config{
		Logging: config.LoggingConfig{Level: "invalid-level"},
	}

	logger, err := InitLogger(cfg)

	require.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "invalid logging level")
}
