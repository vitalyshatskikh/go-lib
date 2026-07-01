package postgres

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
)

// NewPGXPool creates a new observable connection pool.
//
// PoolName configures the logical pool identity used as the
// "db.client.connection.pool.name" trace attribute and the "client_pool_name"
// Prometheus label. If PoolName is empty (default), it is derived from the
// "{db-name}-{target-session-attrs}" template.
func NewPGXPool(cfg config.PostgresConfig, logger *zap.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.ConnString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}
	if cfg.HealthCheckPeriod > 0 {
		poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	}

	if cfg.PingTimeout > 0 {
		poolCfg.PingTimeout = cfg.PingTimeout
	}
	if cfg.ConnectTimeout > 0 {
		poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	}

	poolName := cfg.PoolName
	if poolName == "" {
		poolName = poolCfg.ConnConfig.Database
	}

	otelTracer := otelpgx.NewTracer(
		otelpgx.WithTracerAttributes(semconv.DBClientConnectionPoolName(poolName)),
		otelpgx.WithMeterAttributes(semconv.DBClientConnectionPoolName(poolName)),
	)
	if cfg.SlowQueryThreshold > 0 {
		poolCfg.ConnConfig.Tracer = &slowQueryTracer{
			inner:     otelTracer,
			threshold: cfg.SlowQueryThreshold,
			logger:    logger,
		}
	} else {
		poolCfg.ConnConfig.Tracer = otelTracer
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	registerPGXPoolMetrics(pool, poolName)

	return pool, nil
}
