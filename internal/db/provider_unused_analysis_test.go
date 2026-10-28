package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAnalyzeUnusedMetrics(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()
	now := time.Now().UTC()

	mustUpsertCatalog(t, p, []MetricCatalogItem{{Name: "m1"}, {Name: "m2"}, {Name: "m3"}, {Name: "stale"}})
	b.markCatalogStale(t, p, "stale")
	mustUpsertJobIndex(t, p, []MetricJobIndexItem{
		{Name: "m1", Job: "api"}, {Name: "m2", Job: "api"}, {Name: "m3", Job: "node"}, {Name: "stale", Job: "node"},
	})
	require.NoError(t, p.UpsertMetricsCatalogSources(ctx, "source-a", []string{"m1", "m3", "stale"}, now))
	require.NoError(t, p.UpsertMetricsCatalogSources(ctx, "source-b", []string{"m2"}, now))
	mustInsertQueries(t, p, []Query{instantQuery(now.Add(-5*time.Minute), "m1")}) // m1 is used
	require.NoError(t, p.RefreshMetricsUsageSummary(ctx, presenceWindow(t, now.Add(-time.Hour))))

	analyze := func(params UnusedAnalysisParams) []UnusedAnalysisRow {
		t.Helper()
		params.SyncedSince = now.Add(-24 * time.Hour)
		rows, err := p.AnalyzeUnusedMetrics(ctx, params)
		require.NoError(t, err)
		return rows
	}

	// The stale metric's sync is older than the window, so it isn't counted.
	assert.Equal(t, []UnusedAnalysisRow{
		{Job: "api", Total: 2, Unused: 1},
		{Job: "node", Total: 1, Unused: 1},
	}, analyze(UnusedAnalysisParams{GroupBy: UnusedAnalysisByJob}))

	assert.Equal(t, []UnusedAnalysisRow{
		{Source: "source-a", Total: 2, Unused: 1},
		{Source: "source-b", Total: 1, Unused: 1},
	}, analyze(UnusedAnalysisParams{GroupBy: UnusedAnalysisBySource}))

	assert.Equal(t, []UnusedAnalysisRow{
		{Job: "api", Source: "source-b", Total: 1, Unused: 1},
		{Job: "node", Source: "source-a", Total: 1, Unused: 1},
		{Job: "api", Source: "source-a", Total: 1, Unused: 0},
	}, analyze(UnusedAnalysisParams{GroupBy: UnusedAnalysisByJobSource}))

	assert.Equal(t, []UnusedAnalysisRow{
		{Job: "api", Source: "source-a", Total: 1, Unused: 0},
	}, analyze(UnusedAnalysisParams{GroupBy: UnusedAnalysisByJobSource, Job: "api", Source: "source-a"}))

	_, err := p.AnalyzeUnusedMetrics(ctx, UnusedAnalysisParams{GroupBy: "pod"})
	assert.Error(t, err)
}

func TestPostgreSQL_AnalyzeUnusedMetrics(t *testing.T) {
	t.Parallel()
	testAnalyzeUnusedMetrics(t, postgreSQLTestBackend)
}

func TestSQLite_AnalyzeUnusedMetrics(t *testing.T) {
	t.Parallel()
	testAnalyzeUnusedMetrics(t, sqliteTestBackend)
}
