package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_WhenEnvVarsSet_ThenPopulatesConfig(t *testing.T) {
	t.Setenv("APP_NAME", "test-app")
	t.Setenv("APP_VERSION", "2.0.0")
	t.Setenv("APP_ENVIRONMENT", "testing")
	t.Setenv("API_HOST", "127.0.0.1")
	t.Setenv("API_PORT", "9090")
	t.Setenv("METRICS_ENABLED", "false")
	t.Setenv("LOGGING_LEVEL", "debug")
	t.Setenv("DEBUG", "true")

	cfg, err := Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "test-app", cfg.App.Name)
	assert.Equal(t, "2.0.0", cfg.App.Version)
	assert.Equal(t, "testing", cfg.App.Environment)
	assert.Equal(t, "127.0.0.1", cfg.API.Host)
	assert.Equal(t, 9090, cfg.API.Port)
	assert.False(t, cfg.Metrics.Enabled)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.True(t, cfg.Debug)
}

func TestLoad_WhenNoEnvVarsSet_ThenReturnsDefaults(t *testing.T) {
	cfg, err := Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "my-app", cfg.App.Name)
	assert.Equal(t, "0.1.0", cfg.App.Version)
	assert.Equal(t, "development", cfg.App.Environment)
	assert.Equal(t, "0.0.0.0", cfg.API.Host)
	assert.Equal(t, 8080, cfg.API.Port)
	assert.True(t, cfg.Metrics.Enabled)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.False(t, cfg.Debug)
}

func TestLoadFromEnvFile_WhenFileExists_ThenPopulatesConfig(t *testing.T) {
	content := `APP_NAME=envfile-app
APP_VERSION=3.0.0
API_PORT=7070
DEBUG=true
`
	envFile := filepath.Join(t.TempDir(), ".env")
	err := os.WriteFile(envFile, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromEnvFile(envFile)

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "envfile-app", cfg.App.Name)
	assert.Equal(t, "3.0.0", cfg.App.Version)
	assert.Equal(t, 7070, cfg.API.Port)
	assert.True(t, cfg.Debug)
}

func TestLoadFromEnvFile_WhenFileMissing_ThenReturnsError(t *testing.T) {
	cfg, err := LoadFromEnvFile("/nonexistent/path/.env")

	assert.Error(t, err)
	assert.Nil(t, cfg)
}
