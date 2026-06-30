package postgres

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
)

// NewPGXPool creates new observable connection pool
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

	otelTracer := otelpgx.NewTracer()
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

	registerPGXPoolMetrics(pool)

	return pool, nil
}
