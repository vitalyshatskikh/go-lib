package postgres

import (
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type poolKey struct {
	dbName   string
	poolName string
}

var (
	pgxpoolCollector  = newPgxpoolCollector()
	registerCollector sync.Once
)

type poolCollector struct {
	pools sync.Map // keyed by poolKey -> *pgxpool.Pool

	maxConnsDesc          *prometheus.Desc
	totalConnsDesc        *prometheus.Desc
	acquiredConnsDesc     *prometheus.Desc
	idleConnsDesc         *prometheus.Desc
	constructingConnsDesc *prometheus.Desc
	acquireTotalDesc      *prometheus.Desc
	acquireDurationDesc   *prometheus.Desc
}

func newPgxpoolCollector() *poolCollector {
	labels := []string{"db_name", "client_pool_name"}
	constLabels := prometheus.Labels{"type": "pgxpool"}

	return &poolCollector{
		maxConnsDesc: prometheus.NewDesc(
			prometheus.BuildFQName("", "pg_pool", "max_conns"),
			"Maximum pool size.",
			labels, constLabels,
		),
		totalConnsDesc: prometheus.NewDesc(
			prometheus.BuildFQName("", "pg_pool", "total_conns"),
			"Total number of resources in the pool.",
			labels, constLabels,
		),
		acquiredConnsDesc: prometheus.NewDesc(
			prometheus.BuildFQName("", "pg_pool", "acquired_conns"),
			"Currently acquired connections.",
			labels, constLabels,
		),
		idleConnsDesc: prometheus.NewDesc(
			prometheus.BuildFQName("", "pg_pool", "idle_conns"),
			"Currently idle connections.",
			labels, constLabels,
		),
		constructingConnsDesc: prometheus.NewDesc(
			prometheus.BuildFQName("", "pg_pool", "constructing_conns"),
			"Connections being constructed.",
			labels, constLabels,
		),
		acquireTotalDesc: prometheus.NewDesc(
			prometheus.BuildFQName("", "pg_pool", "acquire_total"),
			"Cumulative number of successful acquires.",
			labels, constLabels,
		),
		acquireDurationDesc: prometheus.NewDesc(
			prometheus.BuildFQName("", "pg_pool", "acquire_duration_seconds_total"),
			"Cumulative total time spent acquiring connections in seconds.",
			labels, constLabels,
		),
	}
}

func (c *poolCollector) add(pool *pgxpool.Pool, poolName string) {
	dbName := pool.Config().ConnConfig.Database
	c.pools.Store(poolKey{dbName: dbName, poolName: poolName}, pool)
}

func (c *poolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.maxConnsDesc
	ch <- c.totalConnsDesc
	ch <- c.acquiredConnsDesc
	ch <- c.idleConnsDesc
	ch <- c.constructingConnsDesc
	ch <- c.acquireTotalDesc
	ch <- c.acquireDurationDesc
}

func (c *poolCollector) Collect(ch chan<- prometheus.Metric) {
	c.pools.Range(func(key, value any) bool {
		pool, ok := value.(*pgxpool.Pool)
		if !ok {
			return true
		}
		pk, ok := key.(poolKey)
		if !ok {
			return true
		}
		stats := pool.Stat()

		ch <- prometheus.MustNewConstMetric(
			c.maxConnsDesc, prometheus.GaugeValue, float64(stats.MaxConns()), pk.dbName, pk.poolName,
		)
		ch <- prometheus.MustNewConstMetric(
			c.totalConnsDesc, prometheus.GaugeValue, float64(stats.TotalConns()), pk.dbName, pk.poolName,
		)
		ch <- prometheus.MustNewConstMetric(
			c.acquiredConnsDesc, prometheus.GaugeValue, float64(stats.AcquiredConns()), pk.dbName, pk.poolName,
		)
		ch <- prometheus.MustNewConstMetric(
			c.idleConnsDesc, prometheus.GaugeValue, float64(stats.IdleConns()), pk.dbName, pk.poolName,
		)
		ch <- prometheus.MustNewConstMetric(
			c.constructingConnsDesc, prometheus.GaugeValue, float64(stats.ConstructingConns()), pk.dbName, pk.poolName,
		)
		ch <- prometheus.MustNewConstMetric(
			c.acquireTotalDesc, prometheus.CounterValue, float64(stats.AcquireCount()), pk.dbName, pk.poolName,
		)
		ch <- prometheus.MustNewConstMetric(
			c.acquireDurationDesc, prometheus.CounterValue, stats.AcquireDuration().Seconds(), pk.dbName, pk.poolName,
		)

		return true
	})
}

func registerPGXPoolMetrics(pool *pgxpool.Pool, poolName string) {
	registerCollector.Do(func() {
		prometheus.MustRegister(pgxpoolCollector)
	})
	pgxpoolCollector.add(pool, poolName)
}
