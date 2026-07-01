//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vitalyshatskikh/go-lib/config"
)

func TestPoolCollector_Collect_WhenPool_ThenEmitsPoolMetrics(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	pool, err := pgxpool.New(context.Background(), cfg.Postgres.ConnString())
	require.NoError(t, err)
	defer pool.Close()

	c := newPgxpoolCollector()
	c.add(pool, "test-pool")

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(c)

	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 7)

	names := make([]string, 0, 7)
	for _, f := range families {
		names = append(names, f.GetName())
	}
	assert.ElementsMatch(t, []string{
		"pg_pool_max_conns",
		"pg_pool_total_conns",
		"pg_pool_acquired_conns",
		"pg_pool_idle_conns",
		"pg_pool_constructing_conns",
		"pg_pool_acquire_total",
		"pg_pool_acquire_duration_seconds_total",
	}, names)
}

func TestPoolCollector_Collect_WhenCalled_ThenPoolsHaveDbNameLabel(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	pool, err := pgxpool.New(context.Background(), cfg.Postgres.ConnString())
	require.NoError(t, err)
	defer pool.Close()

	c := newPgxpoolCollector()
	c.add(pool, "test-pool")

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(c)

	families, err := reg.Gather()
	require.NoError(t, err)

	for _, f := range families {
		for _, m := range f.GetMetric() {
			hasDbName := false
			hasClientPoolName := false
			for _, l := range m.GetLabel() {
				if l.GetName() == "db_name" && l.GetValue() == "postgres" {
					hasDbName = true
				}
				if l.GetName() == "client_pool_name" && l.GetValue() == "test-pool" {
					hasClientPoolName = true
				}
			}
			assert.True(t, hasDbName, "metric %s missing db_name=postgres label", f.GetName())
			assert.True(t, hasClientPoolName, "metric %s missing client_pool_name=test-pool label", f.GetName())
		}
	}
}

func TestPoolCollector_Collect_WhenMultiplePoolsWithSameDbNameAndDifferentPoolName_ThenBothEmitted(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	pool1, err := pgxpool.New(context.Background(), cfg.Postgres.ConnString())
	require.NoError(t, err)
	defer pool1.Close()

	pool2, err := pgxpool.New(context.Background(), cfg.Postgres.ConnString())
	require.NoError(t, err)
	defer pool2.Close()

	c := newPgxpoolCollector()
	c.add(pool1, "pool-alpha")
	c.add(pool2, "pool-beta")

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(c)

	families, err := reg.Gather()
	require.NoError(t, err)

	require.Len(t, families, 7)
	for _, f := range families {
		require.Len(t, f.GetMetric(), 2, "expected 2 metric entries for %s (one per pool name)", f.GetName())
	}
}
