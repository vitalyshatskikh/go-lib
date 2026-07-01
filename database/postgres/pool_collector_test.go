package postgres

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestPoolCollector_Describe_WhenCalled_ThenEmitsSevenDescs(t *testing.T) {
	c := newPgxpoolCollector()
	descs := make(chan *prometheus.Desc, 7)

	c.Describe(descs)
	close(descs)

	count := 0
	for range descs {
		count++
	}
	assert.Equal(t, 7, count)
}

func TestPoolCollector_Describe_WhenCalled_ThenDescsHaveClientPoolNameLabel(t *testing.T) {
	c := newPgxpoolCollector()
	descs := make(chan *prometheus.Desc, 7)
	c.Describe(descs)
	close(descs)

	for desc := range descs {
		s := desc.String()
		assert.Contains(t, s, "client_pool_name")
		assert.Contains(t, s, "db_name")
	}
}

func TestPoolCollector_Collect_WhenNoPools_ThenEmitsNoMetrics(t *testing.T) {
	c := newPgxpoolCollector()
	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(c)

	families, err := reg.Gather()
	assert.NoError(t, err)
	assert.Empty(t, families)
}
