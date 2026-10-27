package db

import (
	"context"
	"testing"
	"time"

	"github.com/nicolastakashi/prom-analytics-proxy/api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSeriesMetadataSourceFilter checks the source filter narrows every
// shape of the catalog query: all, used, unused and unused within a job.
func testSeriesMetadataSourceFilter(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()
	now := time.Now().UTC()

	mustUpsertCatalog(t, p, []MetricCatalogItem{{Name: "m1"}, {Name: "m2"}, {Name: "m3"}, {Name: "m4"}})
	mustUpsertJobIndex(t, p, []MetricJobIndexItem{{Name: "m1", Job: "a"}, {Name: "m2", Job: "a"}, {Name: "m3", Job: "b"}})
	require.NoError(t, p.UpsertMetricsCatalogSources(ctx, "source-a", []string{"m1", "m3"}, now))
	require.NoError(t, p.UpsertMetricsCatalogSources(ctx, "source-b", []string{"m2", "m3"}, now))
	mustInsertQueries(t, p, []Query{instantQuery(now.Add(-5*time.Minute), "m3")}) // m3 is used
	require.NoError(t, p.RefreshMetricsUsageSummary(ctx, presenceWindow(t, now.Add(-time.Hour))))

	names := func(params SeriesMetadataParams) []string {
		t.Helper()
		params.Page, params.PageSize, params.Type = 1, 10, "all"
		params.SortBy, params.SortOrder = "name", "asc"
		res, err := p.GetSeriesMetadata(ctx, params)
		require.NoError(t, err)
		return keysOf(rowsOf[models.MetricMetadata](t, res), func(m models.MetricMetadata) string { return m.Name })
	}

	assert.Equal(t, []string{"m1", "m2", "m3", "m4"}, names(SeriesMetadataParams{Usage: SeriesMetadataUsageAll}), "no source filter")
	assert.Equal(t, []string{"m1", "m3"}, names(SeriesMetadataParams{Usage: SeriesMetadataUsageAll, Source: "source-a"}))
	assert.Equal(t, []string{"m3"}, names(SeriesMetadataParams{Usage: SeriesMetadataUsageUsed, Source: "source-b"}))
	assert.Equal(t, []string{"m2"}, names(SeriesMetadataParams{Usage: SeriesMetadataUsageUnused, Source: "source-b"}))
	assert.Equal(t, []string{"m1"}, names(SeriesMetadataParams{Usage: SeriesMetadataUsageUnused, Job: "a", Source: "source-a"}))
	assert.Empty(t, names(SeriesMetadataParams{Usage: SeriesMetadataUsageAll, Source: "unknown"}))

	require.NoError(t, p.UpsertMetricsServiceIndex(ctx, []MetricServiceIndexItem{
		{Name: "m1", Service: "billing", Job: "a"}, {Name: "m2", Service: "payments", Job: "a"},
	}, time.Now()))
	assert.Equal(t, []string{"m1"}, names(SeriesMetadataParams{Usage: SeriesMetadataUsageAll, Service: "billing"}))
	assert.Equal(t, []string{"m2"}, names(SeriesMetadataParams{Usage: SeriesMetadataUsageUnused, Service: "payments"}))
	assert.Equal(t, []string{"m1"}, names(SeriesMetadataParams{Usage: SeriesMetadataUsageUnused, Job: "a", Service: "billing"}))
	assert.Empty(t, names(SeriesMetadataParams{Usage: SeriesMetadataUsageAll, Service: "billing", Source: "source-b"}))
}

func TestPostgreSQL_GetSeriesMetadata_SourceFilter(t *testing.T) {
	t.Parallel()
	testSeriesMetadataSourceFilter(t, postgreSQLTestBackend)
}

func TestSQLite_GetSeriesMetadata_SourceFilter(t *testing.T) {
	t.Parallel()
	testSeriesMetadataSourceFilter(t, sqliteTestBackend)
}
