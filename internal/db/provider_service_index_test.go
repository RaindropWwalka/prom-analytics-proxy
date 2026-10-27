package db

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	}, time.Now()))
	require.NoError(t, p.UpsertMetricsServiceIndex(ctx, []MetricServiceIndexItem{item}, time.Now())) // and again later
	require.NoError(t, p.UpsertMetricsServiceIndex(ctx, nil, time.Now()))

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

func testMetricsServiceIndexRefreshAndChunks(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()
	now := time.Now()

	// More pairs than one statement writes.
	var many []MetricServiceIndexItem
	for i := range serviceIndexChunk + 1000 {
		many = append(many, MetricServiceIndexItem{Name: fmt.Sprintf("m%05d", i), Service: "billing", Job: "jmx"})
	}
	require.NoError(t, p.UpsertMetricsServiceIndex(ctx, many, now))
	var rows int
	require.NoError(t, rawDB(p).QueryRowContext(ctx, `SELECT COUNT(*) FROM metrics_service_index`).Scan(&rows))
	assert.Equal(t, len(many), rows, "every chunk lands")

	stamp := func(name string, at time.Time) {
		b.mustExec(t, p, `UPDATE metrics_service_index SET updated_at = ? WHERE name = ?`, at.UTC().Format(SQLiteTimeFormat), name)
	}
	seenBefore := func(name string, at time.Time) bool {
		var n int
		require.NoError(t, rawDB(p).QueryRowContext(ctx,
			b.rebind(`SELECT COUNT(*) FROM metrics_service_index WHERE name = ? AND updated_at < ?`),
			name, at.UTC().Format(SQLiteTimeFormat)).Scan(&n))
		return n == 1
	}
	stamp("m00000", now.Add(-2*time.Hour))
	stamp("m00001", now.Add(-48*time.Hour))

	require.NoError(t, p.UpsertMetricsServiceIndex(ctx, many[:2], now.Add(-24*time.Hour)))

	assert.True(t, seenBefore("m00000", now.Add(-time.Hour)), "a pair confirmed within the refresh interval isn't rewritten")
	assert.False(t, seenBefore("m00001", now.Add(-time.Hour)), "an older pair is refreshed")
}

func TestPostgreSQL_MetricsServiceIndex_RefreshAndChunks(t *testing.T) {
	t.Parallel()
	testMetricsServiceIndexRefreshAndChunks(t, postgreSQLTestBackend)
}

func TestSQLite_MetricsServiceIndex_RefreshAndChunks(t *testing.T) {
	t.Parallel()
	testMetricsServiceIndexRefreshAndChunks(t, sqliteTestBackend)
}
