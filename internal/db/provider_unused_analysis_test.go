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

	require.NoError(t, p.UpsertMetricsServiceIndex(ctx, []MetricServiceIndexItem{
		{Name: "m1", Service: "billing", Job: "api"},
		{Name: "m2", Service: "billing", Job: "api"},
		{Name: "m2", Service: "billing", Job: "other"}, // counts once for its service
		{Name: "m3", Service: "metrics", Job: "node"},
	}))
	assert.Equal(t, []UnusedAnalysisRow{
		{Service: "billing", Total: 2, Unused: 1},
		{Service: "metrics", Total: 1, Unused: 1},
	}, analyze(UnusedAnalysisParams{GroupBy: UnusedAnalysisByService}))
	assert.Equal(t, []UnusedAnalysisRow{
		{Service: "billing", Source: "source-b", Total: 1, Unused: 1},
		{Service: "metrics", Source: "source-a", Total: 1, Unused: 1},
		{Service: "billing", Source: "source-a", Total: 1, Unused: 0},
	}, analyze(UnusedAnalysisParams{GroupBy: UnusedAnalysisByServiceSource}))

	assert.Equal(t, []UnusedAnalysisRow{
		{Job: "api", Service: "billing", Total: 2, Unused: 1},
		{Job: "node", Service: "metrics", Total: 1, Unused: 1},
		{Job: "other", Service: "billing", Total: 1, Unused: 1},
	}, analyze(UnusedAnalysisParams{GroupBy: UnusedAnalysisByJobService}))
	assert.Equal(t, []UnusedAnalysisRow{
		{Job: "api", Service: "billing", Source: "source-a", Total: 1, Unused: 0},
	}, analyze(UnusedAnalysisParams{GroupBy: UnusedAnalysisByJobServiceSource, Job: "api", Source: "source-a"}))

	// Series: m2 (unused) is the heavy one; m1 (used) is light.
	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-a", map[string]int{"m1": 5, "m3": 7}))
	require.NoError(t, p.ReplaceMetricsCardinality(ctx, "source-b", map[string]int{"m2": 100}))
	assert.Equal(t, []UnusedAnalysisRow{
		{Job: "api", Total: 2, Unused: 1, Series: 105, UnusedSeries: 100},
		{Job: "node", Total: 1, Unused: 1, Series: 7, UnusedSeries: 7},
	}, analyze(UnusedAnalysisParams{GroupBy: UnusedAnalysisByJob}))
	assert.Equal(t, []UnusedAnalysisRow{
		{Job: "api", Source: "source-b", Total: 1, Unused: 1, Series: 100, UnusedSeries: 100},
		{Job: "node", Source: "source-a", Total: 1, Unused: 1, Series: 7, UnusedSeries: 7},
		{Job: "api", Source: "source-a", Total: 1, Unused: 0, Series: 5, UnusedSeries: 0},
	}, analyze(UnusedAnalysisParams{GroupBy: UnusedAnalysisByJobSource}), "per source, a source's own series")

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
