package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/nicolastakashi/prom-analytics-proxy/api/models"
)

// ErrDatabaseUnavailable is what a DeferredProvider returns until the
// database it stands in for is connected.
var ErrDatabaseUnavailable = errors.New("database is not connected yet")

// DeferredProvider stands in for a Provider whose database wasn't reachable
// at startup (database.connect_in_background): every call fails with
// ErrDatabaseUnavailable until Set hands it the connected Provider, then
// passes through. The proxy keeps serving queries in the meantime; only what
// it records about them is lost.
type DeferredProvider struct {
	mu sync.RWMutex
	p  Provider
}

var _ Provider = (*DeferredProvider)(nil)

func NewDeferredProvider() *DeferredProvider { return &DeferredProvider{} }

// Set makes p the Provider every later call goes to.
func (d *DeferredProvider) Set(p Provider) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.p = p
}

func (d *DeferredProvider) current() (Provider, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.p == nil {
		return nil, ErrDatabaseUnavailable
	}
	return d.p, nil
}

// WithDB calls fn only once connected.
func (d *DeferredProvider) WithDB(fn func(db *sql.DB)) {
	if p, err := d.current(); err == nil {
		p.WithDB(fn)
	}
}

func (d *DeferredProvider) Close() error {
	if p, err := d.current(); err == nil {
		return p.Close()
	}
	return nil
}

// ConnectWithRetry calls connect until it succeeds, waiting between attempts
// from initialBackoff doubling up to maxBackoff. It gives up only when ctx is
// done.
func ConnectWithRetry(ctx context.Context, connect func(context.Context) (Provider, error), initialBackoff, maxBackoff time.Duration) (Provider, error) {
	backoff := initialBackoff
	for {
		p, err := connect(ctx)
		if err == nil {
			return p, nil
		}
		slog.Warn("database still unavailable, retrying", "in", backoff, "err", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff = min(2*backoff, maxBackoff)
	}
}

func (d *DeferredProvider) Insert(ctx context.Context, queries []Query) error {
	p, err := d.current()
	if err != nil {
		return err
	}
	return p.Insert(ctx, queries)
}

func (d *DeferredProvider) InsertRulesUsage(ctx context.Context, rulesUsage []RulesUsage) error {
	p, err := d.current()
	if err != nil {
		return err
	}
	return p.InsertRulesUsage(ctx, rulesUsage)
}

func (d *DeferredProvider) InsertDashboardUsage(ctx context.Context, dashboardUsage []DashboardUsage) error {
	p, err := d.current()
	if err != nil {
		return err
	}
	return p.InsertDashboardUsage(ctx, dashboardUsage)
}

func (d *DeferredProvider) GetSeriesMetadata(ctx context.Context, params SeriesMetadataParams) (PagedResult, error) {
	p, err := d.current()
	if err != nil {
		return PagedResult{}, err
	}
	return p.GetSeriesMetadata(ctx, params)
}

func (d *DeferredProvider) UpsertMetricsCatalog(ctx context.Context, items []MetricCatalogItem) error {
	p, err := d.current()
	if err != nil {
		return err
	}
	return p.UpsertMetricsCatalog(ctx, items)
}

func (d *DeferredProvider) RefreshMetricsUsageSummary(ctx context.Context, tr TimeRange) error {
	p, err := d.current()
	if err != nil {
		return err
	}
	return p.RefreshMetricsUsageSummary(ctx, tr)
}

func (d *DeferredProvider) UpsertMetricsJobIndex(ctx context.Context, items []MetricJobIndexItem) error {
	p, err := d.current()
	if err != nil {
		return err
	}
	return p.UpsertMetricsJobIndex(ctx, items)
}

func (d *DeferredProvider) ListJobs(ctx context.Context) ([]string, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.ListJobs(ctx)
}

func (d *DeferredProvider) UpsertMetricsServiceIndex(ctx context.Context, items []MetricServiceIndexItem, refreshBefore time.Time) error {
	p, err := d.current()
	if err != nil {
		return err
	}
	return p.UpsertMetricsServiceIndex(ctx, items, refreshBefore)
}

func (d *DeferredProvider) ListServices(ctx context.Context) ([]string, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.ListServices(ctx)
}

func (d *DeferredProvider) UpsertMetricsCatalogSources(ctx context.Context, source string, names []string, refreshBefore time.Time) error {
	p, err := d.current()
	if err != nil {
		return err
	}
	return p.UpsertMetricsCatalogSources(ctx, source, names, refreshBefore)
}

func (d *DeferredProvider) DeleteMetricsCatalogSourcesBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	p, err := d.current()
	if err != nil {
		return 0, err
	}
	return p.DeleteMetricsCatalogSourcesBefore(ctx, cutoff)
}

func (d *DeferredProvider) GetMetricsCatalogSources(ctx context.Context, names []string) (map[string][]string, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetMetricsCatalogSources(ctx, names)
}

func (d *DeferredProvider) ListMetricsCatalogSources(ctx context.Context) ([]string, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.ListMetricsCatalogSources(ctx)
}

func (d *DeferredProvider) ReplaceMetricsCardinality(ctx context.Context, source string, series map[string]int, refreshBefore time.Time) error {
	p, err := d.current()
	if err != nil {
		return err
	}
	return p.ReplaceMetricsCardinality(ctx, source, series, refreshBefore)
}

func (d *DeferredProvider) GetMetricsCardinality(ctx context.Context, names []string) (map[string]map[string]int, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetMetricsCardinality(ctx, names)
}

func (d *DeferredProvider) AnalyzeUnusedMetrics(ctx context.Context, params UnusedAnalysisParams) ([]UnusedAnalysisRow, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.AnalyzeUnusedMetrics(ctx, params)
}

func (d *DeferredProvider) GetQueryTypes(ctx context.Context, tr TimeRange, fingerprint string) (*QueryTypesResult, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetQueryTypes(ctx, tr, fingerprint)
}

func (d *DeferredProvider) GetAverageDuration(ctx context.Context, tr TimeRange, fingerprint string) (*AverageDurationResult, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetAverageDuration(ctx, tr, fingerprint)
}

func (d *DeferredProvider) GetQueryRate(ctx context.Context, tr TimeRange, metricName string, fingerprint string) (*QueryRateResult, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetQueryRate(ctx, tr, metricName, fingerprint)
}

func (d *DeferredProvider) GetQueriesBySerieName(ctx context.Context, params QueriesBySerieNameParams) (PagedResult, error) {
	p, err := d.current()
	if err != nil {
		return PagedResult{}, err
	}
	return p.GetQueriesBySerieName(ctx, params)
}

func (d *DeferredProvider) GetQueryStatusDistribution(ctx context.Context, tr TimeRange, fingerprint string) ([]QueryStatusDistributionResult, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetQueryStatusDistribution(ctx, tr, fingerprint)
}

func (d *DeferredProvider) GetQueryLatencyTrends(ctx context.Context, tr TimeRange, metricName string, fingerprint string) ([]QueryLatencyTrendsResult, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetQueryLatencyTrends(ctx, tr, metricName, fingerprint)
}

func (d *DeferredProvider) GetQueryThroughputAnalysis(ctx context.Context, tr TimeRange) ([]QueryThroughputAnalysisResult, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetQueryThroughputAnalysis(ctx, tr)
}

func (d *DeferredProvider) GetQueryErrorAnalysis(ctx context.Context, tr TimeRange, fingerprint string) ([]QueryErrorAnalysisResult, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetQueryErrorAnalysis(ctx, tr, fingerprint)
}

func (d *DeferredProvider) GetQueryTimeRangeDistribution(ctx context.Context, tr TimeRange, fingerprint string) ([]QueryTimeRangeDistributionResult, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetQueryTimeRangeDistribution(ctx, tr, fingerprint)
}

func (d *DeferredProvider) GetQueryExpressions(ctx context.Context, params QueryExpressionsParams) (PagedResult, error) {
	p, err := d.current()
	if err != nil {
		return PagedResult{}, err
	}
	return p.GetQueryExpressions(ctx, params)
}

func (d *DeferredProvider) GetQueryExecutions(ctx context.Context, params QueryExecutionsParams) (PagedResult, error) {
	p, err := d.current()
	if err != nil {
		return PagedResult{}, err
	}
	return p.GetQueryExecutions(ctx, params)
}

func (d *DeferredProvider) GetMetricStatistics(ctx context.Context, metricName string, tr TimeRange) (MetricUsageStatics, error) {
	p, err := d.current()
	if err != nil {
		return MetricUsageStatics{}, err
	}
	return p.GetMetricStatistics(ctx, metricName, tr)
}

func (d *DeferredProvider) GetMetricQueryPerformanceStatistics(ctx context.Context, metricName string, tr TimeRange) (MetricQueryPerformanceStatistics, error) {
	p, err := d.current()
	if err != nil {
		return MetricQueryPerformanceStatistics{}, err
	}
	return p.GetMetricQueryPerformanceStatistics(ctx, metricName, tr)
}

func (d *DeferredProvider) GetRulesUsage(ctx context.Context, params RulesUsageParams) (PagedResult, error) {
	p, err := d.current()
	if err != nil {
		return PagedResult{}, err
	}
	return p.GetRulesUsage(ctx, params)
}

func (d *DeferredProvider) GetDashboardUsage(ctx context.Context, params DashboardUsageParams) (PagedResult, error) {
	p, err := d.current()
	if err != nil {
		return PagedResult{}, err
	}
	return p.GetDashboardUsage(ctx, params)
}

func (d *DeferredProvider) GetSeriesMetadataByNames(ctx context.Context, names []string, job string) ([]models.MetricMetadata, error) {
	p, err := d.current()
	if err != nil {
		return nil, err
	}
	return p.GetSeriesMetadataByNames(ctx, names, job)
}

func (d *DeferredProvider) DeleteQueriesBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	p, err := d.current()
	if err != nil {
		return 0, err
	}
	return p.DeleteQueriesBefore(ctx, cutoff)
}

func (d *DeferredProvider) GetJobLastRun(ctx context.Context, name string) (time.Time, bool, error) {
	p, err := d.current()
	if err != nil {
		return time.Time{}, false, err
	}
	return p.GetJobLastRun(ctx, name)
}

func (d *DeferredProvider) SetJobLastRun(ctx context.Context, name string, at time.Time) error {
	p, err := d.current()
	if err != nil {
		return err
	}
	return p.SetJobLastRun(ctx, name, at)
}
