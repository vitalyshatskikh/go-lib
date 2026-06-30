//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
	"github.com/vitalyshatskikh/go-lib/database/postgres"
)

func TestNewPGXPool_WhenConfigIsValid_ThenConfigApplied(t *testing.T) {
	cfg := config.PostgresConfig{
		Hosts:              []string{"localhost:15432"},
		User:               "postgres",
		Password:           "postgres",
		Database:           "postgres",
		SSLMode:            "disable",
		TargetSessionAttrs: "primary",
		MaxConns:           5,
		MinConns:           2,
		MaxConnLifetime:    30 * time.Minute,
		MaxConnIdleTime:    5 * time.Minute,
		HealthCheckPeriod:  30 * time.Second,
	}

	pool, err := postgres.NewPGXPool(cfg, zap.NewNop())
	require.NoError(t, err)
	defer pool.Close()

	c := pool.Config()
	assert.Equal(t, int32(5), c.MaxConns)
	assert.Equal(t, int32(2), c.MinConns)
	assert.Equal(t, 30*time.Minute, c.MaxConnLifetime)
	assert.Equal(t, 5*time.Minute, c.MaxConnIdleTime)
	assert.Equal(t, 30*time.Second, c.HealthCheckPeriod)
}

func TestNewPGXPool_WhenSlowQueryThresholdSet_ThenPoolIsCreated(t *testing.T) {
	cfg := config.PostgresConfig{
		Hosts:              []string{"localhost:15432"},
		User:               "postgres",
		Password:           "postgres",
		Database:           "postgres",
		SSLMode:            "disable",
		TargetSessionAttrs: "primary",
		MaxConns:           5,
		MaxConnLifetime:    1 * time.Hour,
		MaxConnIdleTime:    30 * time.Minute,
		HealthCheckPeriod:  1 * time.Minute,
		SlowQueryThreshold: 100 * time.Millisecond,
	}

	pool, err := postgres.NewPGXPool(cfg, zap.NewNop())
	require.NoError(t, err)
	defer pool.Close()

	assert.NotNil(t, pool)
}

func TestNewPGXPool_WhenConfigIsEmpty_ThenDefaultConfigApplied(t *testing.T) {
	cfg := config.PostgresConfig{
		Hosts:              []string{"localhost:15432"},
		User:               "postgres",
		Password:           "postgres",
		Database:           "postgres",
		SSLMode:            "disable",
		TargetSessionAttrs: "primary",
		// Pool params at Go zero values → pgxpool internal defaults should be used
	}

	pool, err := postgres.NewPGXPool(cfg, zap.NewNop())
	require.NoError(t, err)
	defer pool.Close()

	c := pool.Config()
	assert.Positive(t, int(c.MaxConns))
	assert.Equal(t, int32(0), c.MinConns)
	assert.Equal(t, time.Hour, c.MaxConnLifetime)
	assert.Equal(t, 30*time.Minute, c.MaxConnIdleTime)
	assert.Equal(t, time.Minute, c.HealthCheckPeriod)
}
