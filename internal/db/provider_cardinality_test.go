package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMetricsCardinality(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()

	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-a", map[string]int{"up": 10, "http_requests_total": 500}, time.Now()))
	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-b", map[string]int{"up": 3}, time.Now()))
	// source-a's next top-N: http_requests_total left it, a new metric joined.
	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-a", map[string]int{"up": 12, "jvm_threads": 40}, time.Now()))

	got, err := p.GetMetricsCardinality(ctx, []string{"up", "http_requests_total", "jvm_threads", "unknown"})
	require.NoError(t, err)
	assert.Equal(t, map[string]map[string]int{
		"up":          {"source-a": 12, "source-b": 3},
		"jvm_threads": {"source-a": 40},
	}, got)

	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-b", nil, time.Now()))
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

// testMetricsCardinalityFallsBackToJobIndex: a metric outside every source's
// top-N takes its series summed over the job index, under
// UpstreamSeriesSource; a metric some source counted keeps only that.
func testMetricsCardinalityFallsBackToJobIndex(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()

	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-a", map[string]int{"up": 10}, time.Now()))
	require.NoError(t, p.UpsertMetricsJobIndex(ctx, []MetricJobIndexItem{
		{Name: "up", Job: "job-a", Series: 7},
		{Name: "jvm_threads", Job: "job-a", Series: 30},
		{Name: "jvm_threads", Job: "job-b", Series: 12},
		{Name: "go_goroutines", Job: "job-a"}, // not counted yet
	}))

	got, err := p.GetMetricsCardinality(ctx, []string{"up", "jvm_threads", "go_goroutines"})
	require.NoError(t, err)
	assert.Equal(t, map[string]map[string]int{
		"up":          {"source-a": 10},
		"jvm_threads": {UpstreamSeriesSource: 42},
	}, got)
}

func TestPostgreSQL_MetricsCardinalityFallsBackToJobIndex(t *testing.T) {
	t.Parallel()
	testMetricsCardinalityFallsBackToJobIndex(t, postgreSQLTestBackend)
}

func TestSQLite_MetricsCardinalityFallsBackToJobIndex(t *testing.T) {
	t.Parallel()
	testMetricsCardinalityFallsBackToJobIndex(t, sqliteTestBackend)
}

// testMetricsCardinalitySkipsSteadyRows pins the write saving: a count within
// a tenth of the stored one, written recently, isn't rewritten.
func testMetricsCardinalitySkipsSteadyRows(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()
	now := time.Now()
	stored := func() int {
		got, err := p.GetMetricsCardinality(ctx, []string{"up"})
		require.NoError(t, err)
		return got["up"]["source-a"]
	}
	dayAgo := now.Add(-24 * time.Hour)

	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-a", map[string]int{"up": 100}, dayAgo))
	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-a", map[string]int{"up": 105}, dayAgo))
	assert.Equal(t, 100, stored(), "a 5% move on a fresh row isn't written")

	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-a", map[string]int{"up": 150}, dayAgo))
	assert.Equal(t, 150, stored(), "a bigger move is")

	b.mustExec(t, p, `UPDATE metrics_cardinality SET updated_at = ? WHERE name = ?`,
		now.Add(-48*time.Hour).UTC().Format(SQLiteTimeFormat), "up")
	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-a", map[string]int{"up": 151}, dayAgo))
	assert.Equal(t, 151, stored(), "and so is any move on a row older than the refresh interval")
}

func TestPostgreSQL_MetricsCardinality_SkipsSteadyRows(t *testing.T) {
	t.Parallel()
	testMetricsCardinalitySkipsSteadyRows(t, postgreSQLTestBackend)
}

func TestSQLite_MetricsCardinality_SkipsSteadyRows(t *testing.T) {
	t.Parallel()
	testMetricsCardinalitySkipsSteadyRows(t, sqliteTestBackend)
}
