package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMetricsCardinality(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()

	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-a", map[string]int{"up": 10, "http_requests_total": 500}))
	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-b", map[string]int{"up": 3}))
	// source-a's next top-N: http_requests_total left it, a new metric joined.
	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-a", map[string]int{"up": 12, "jvm_threads": 40}))

	got, err := p.GetMetricsCardinality(ctx, []string{"up", "http_requests_total", "jvm_threads", "unknown"})
	require.NoError(t, err)
	assert.Equal(t, map[string]map[string]int{
		"up":          {"source-a": 12, "source-b": 3},
		"jvm_threads": {"source-a": 40},
	}, got)

	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-b", nil))
	got, err = p.GetMetricsCardinality(ctx, []string{"up"})
	require.NoError(t, err)
	assert.Equal(t, map[string]map[string]int{"up": {"source-a": 12}}, got, "an empty top-N clears the source")
}

func TestPostgreSQL_MetricsCardinality(t *testing.T) {
	t.Parallel()
	testMetricsCardinality(t, postgreSQLTestBackend)
}

func TestSQLite_MetricsCardinality(t *testing.T) {
	t.Parallel()
	testMetricsCardinality(t, sqliteTestBackend)
}
