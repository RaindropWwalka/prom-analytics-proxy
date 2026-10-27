package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setSourceSeenAt moves a (name, source) pair's last confirmation to at,
// bound in the text format both backends parse into their timestamp column.
func (b testBackend) setSourceSeenAt(t testing.TB, p Provider, name, source string, at time.Time) {
	t.Helper()
	b.mustExec(t, p, `UPDATE metrics_catalog_sources SET last_seen_at = ? WHERE name = ? AND source = ?`,
		at.UTC().Format(SQLiteTimeFormat), name, source)
}

// sourceSeenBefore reports whether the pair was last confirmed before at.
func (b testBackend) sourceSeenBefore(t testing.TB, p Provider, name, source string, at time.Time) bool {
	t.Helper()
	var n int
	err := rawDB(p).QueryRowContext(context.Background(),
		b.rebind(`SELECT COUNT(*) FROM metrics_catalog_sources WHERE name = ? AND source = ? AND last_seen_at < ?`),
		name, source, at.UTC().Format(SQLiteTimeFormat)).Scan(&n)
	require.NoError(t, err)
	return n == 1
}

func testMetricsCatalogSourcesRecordWhoReportsWhat(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()
	refresh := time.Now().Add(-24 * time.Hour)

	require.NoError(t, p.UpsertMetricsCatalogSources(ctx, "source-b", []string{"up", "node_load1", "up"}, refresh))
	require.NoError(t, p.UpsertMetricsCatalogSources(ctx, "source-a", []string{"up", "http_requests_total"}, refresh))
	require.NoError(t, p.UpsertMetricsCatalogSources(ctx, "source-a", nil, refresh))

	got, err := p.GetMetricsCatalogSources(ctx, []string{"up", "node_load1", "http_requests_total", "unknown"})
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{
		"up":                  {"source-a", "source-b"},
		"node_load1":          {"source-b"},
		"http_requests_total": {"source-a"},
	}, got)

	sources, err := p.ListMetricsCatalogSources(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"source-a", "source-b"}, sources)

	empty, err := p.GetMetricsCatalogSources(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestPostgreSQL_MetricsCatalogSources_RecordWhoReportsWhat(t *testing.T) {
	t.Parallel()
	testMetricsCatalogSourcesRecordWhoReportsWhat(t, postgreSQLTestBackend)
}

func TestSQLite_MetricsCatalogSources_RecordWhoReportsWhat(t *testing.T) {
	t.Parallel()
	testMetricsCatalogSourcesRecordWhoReportsWhat(t, sqliteTestBackend)
}

// testMetricsCatalogSourcesOnlyRefreshStalePairs pins the write saving: a
// pair confirmed since refreshBefore is left alone, an older one refreshed.
func testMetricsCatalogSourcesOnlyRefreshStalePairs(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()
	now := time.Now()

	require.NoError(t, p.UpsertMetricsCatalogSources(ctx, "source-b", []string{"fresh", "stale"}, now))
	b.setSourceSeenAt(t, p, "fresh", "source-b", now.Add(-2*time.Hour))
	b.setSourceSeenAt(t, p, "stale", "source-b", now.Add(-48*time.Hour))

	require.NoError(t, p.UpsertMetricsCatalogSources(ctx, "source-b", []string{"fresh", "stale"}, now.Add(-24*time.Hour)))

	assert.True(t, b.sourceSeenBefore(t, p, "fresh", "source-b", now.Add(-time.Hour)),
		"a pair confirmed within the refresh interval must not be rewritten")
	assert.False(t, b.sourceSeenBefore(t, p, "stale", "source-b", now.Add(-time.Hour)),
		"a pair older than the refresh interval must be refreshed")
}

func TestPostgreSQL_MetricsCatalogSources_OnlyRefreshStalePairs(t *testing.T) {
	t.Parallel()
	testMetricsCatalogSourcesOnlyRefreshStalePairs(t, postgreSQLTestBackend)
}

func TestSQLite_MetricsCatalogSources_OnlyRefreshStalePairs(t *testing.T) {
	t.Parallel()
	testMetricsCatalogSourcesOnlyRefreshStalePairs(t, sqliteTestBackend)
}

func testDeleteMetricsCatalogSourcesBeforeDropsOnlyStalePairs(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()
	now := time.Now()

	require.NoError(t, p.UpsertMetricsCatalogSources(ctx, "source-b", []string{"gone", "kept"}, now))
	b.setSourceSeenAt(t, p, "gone", "source-b", now.AddDate(0, 0, -40))

	n, err := p.DeleteMetricsCatalogSourcesBefore(ctx, now.AddDate(0, 0, -30))
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	got, err := p.GetMetricsCatalogSources(ctx, []string{"gone", "kept"})
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"kept": {"source-b"}}, got)
}

func TestPostgreSQL_DeleteMetricsCatalogSourcesBefore_DropsOnlyStalePairs(t *testing.T) {
	t.Parallel()
	testDeleteMetricsCatalogSourcesBeforeDropsOnlyStalePairs(t, postgreSQLTestBackend)
}

func TestSQLite_DeleteMetricsCatalogSourcesBefore_DropsOnlyStalePairs(t *testing.T) {
	t.Parallel()
	testDeleteMetricsCatalogSourcesBeforeDropsOnlyStalePairs(t, sqliteTestBackend)
}
