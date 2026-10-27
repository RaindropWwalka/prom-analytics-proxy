package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMetricsServiceIndex(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()

	item := MetricServiceIndexItem{Name: "jvm_memory_used_bytes", Service: "billing", Job: "jmx"}
	require.NoError(t, p.UpsertMetricsServiceIndex(ctx, []MetricServiceIndexItem{
		item, item, // the same pair twice in one call
		{Name: "jvm_memory_used_bytes", Service: "payments", Job: "jmx"},
	}))
	require.NoError(t, p.UpsertMetricsServiceIndex(ctx, []MetricServiceIndexItem{item})) // and again later
	require.NoError(t, p.UpsertMetricsServiceIndex(ctx, nil))

	services, err := p.ListServices(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"billing", "payments"}, services)

	var rows int
	require.NoError(t, rawDB(p).QueryRowContext(ctx, `SELECT COUNT(*) FROM metrics_service_index`).Scan(&rows))
	assert.Equal(t, 2, rows)
}

func TestPostgreSQL_MetricsServiceIndex(t *testing.T) {
	t.Parallel()
	testMetricsServiceIndex(t, postgreSQLTestBackend)
}

func TestSQLite_MetricsServiceIndex(t *testing.T) {
	t.Parallel()
	testMetricsServiceIndex(t, sqliteTestBackend)
}
