package restapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWithLogger_WhenLoggerProvided_ThenSetsLogger(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	testLogger := zap.NewNop()

	srv, err := New(cfg, WithLogger(testLogger))
	require.NoError(t, err)
	require.NotNil(t, srv)

	assert.Same(t, testLogger, srv.logger)
}

func TestWithLogger_WhenLoggerIsNil_ThenKeepsDefaultLogger(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()

	srv, err := New(cfg, WithLogger(nil))
	require.NoError(t, err)
	require.NotNil(t, srv)

	assert.NotNil(t, srv.logger)
}
