package restapi

import (
	"strings"
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

func TestWithOpenAPI_WhenValidJSON_ThenSetsSpec(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	spec := `{"openapi":"3.0.0","info":{"title":"Test"}}`

	srv, err := New(cfg, WithOpenAPI(strings.NewReader(spec)))
	require.NoError(t, err)
	require.NotNil(t, srv)

	assert.JSONEq(t, spec, string(srv.openapiJSON))
}

func TestWithOpenAPI_WhenValidYAML_ThenConvertsToJSON(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	yamlSpec := "openapi: 3.0.0\ninfo:\n  title: Test\n"

	srv, err := New(cfg, WithOpenAPI(strings.NewReader(yamlSpec)))
	require.NoError(t, err)
	require.NotNil(t, srv)

	assert.JSONEq(t, `{"openapi":"3.0.0","info":{"title":"Test"}}`, string(srv.openapiJSON))
}

func TestWithOpenAPI_WhenInvalidSpec_ReturnsError(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()

	_, err := New(cfg, WithOpenAPI(strings.NewReader("key:\n\tvalue")))
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to parse spec")
}

func TestWithOpenAPI_WhenReaderFails_ReturnsError(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()

	_, err := New(cfg, WithOpenAPI(errReader{}))
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read spec")
}
