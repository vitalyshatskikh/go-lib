package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
	"github.com/vitalyshatskikh/go-lib/database/postgres"
)

func TestNewPGXPool_WhenConfigIsInvalid_ThenError(t *testing.T) {
	cfg := config.PostgresConfig{
		DSN: "postgres://user:pass@host:db?sslmode=%gg",
	}

	pool, err := postgres.NewPGXPool(cfg, zap.NewNop())
	require.Error(t, err)
	require.Nil(t, pool)
	assert.ErrorContains(t, err, "failed to parse config")
}
