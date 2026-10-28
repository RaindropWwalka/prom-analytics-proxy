package inventory

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nicolastakashi/prom-analytics-proxy/internal/config"
	"github.com/nicolastakashi/prom-analytics-proxy/internal/db"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	promconfig "github.com/prometheus/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSourcesSyncer(provider *fakeProvider, nameOnly bool, sources ...metadataSource) *Syncer {
	return &Syncer{
		dbProvider:                    provider,
		metadataSources:               sources,
		timeWindow:                    time.Hour,
		metadataSyncEnabled:           true,
		metadataMetricsNameOnly:       nameOnly,
		runTimeout:                    time.Minute,
		metadataStepTimeout:           30 * time.Second,
		summaryStepTimeout:            30 * time.Second,
		syncDuration:                  newHistogram(),
		syncSuccess:                   newCounter(),
		syncFailure:                   newCounter(),
		catalogSummaryMismatch:        newCounter(),
		metadataSourceFailure:         prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_source_failure"}, []string{"source"}),
		catalogSourcesFailure:         newCounter(),
		metadataSourceRefreshInterval: 24 * time.Hour,
	}
}

func catalogByName(items []db.MetricCatalogItem) map[string]db.MetricCatalogItem {
	out := make(map[string]db.MetricCatalogItem, len(items))
	for _, it := range items {
		out[it.Name] = it
	}
	return out
}

func TestSyncer_MetadataSourcesAreMergedIntoOneCatalog(t *testing.T) {
	provider := &fakeProvider{}
	a := &fakePromAPI{meta: map[string][]v1.Metadata{
		"up":             {{Type: "gauge", Help: "from a"}},
		"only_in_a":      {{Type: "counter", Help: "a"}},
		"described_in_b": {},
	}}
	b := &fakePromAPI{meta: map[string][]v1.Metadata{
		"up":             {{Type: "gauge", Help: "from b"}},
		"described_in_b": {{Type: "counter", Help: "b"}},
	}}
	s := newSourcesSyncer(provider, false, metadataSource{name: "a", api: a}, metadataSource{name: "b", api: b})

	s.RunOnce(context.Background())

	require.Equal(t, 1, provider.catalogCommits, "every source should land in a single commit")
	got := catalogByName(provider.catalogItems)
	assert.Len(t, provider.catalogItems, 3, "a metric known to several sources is committed once")
	assert.Equal(t, "from a", got["up"].Help, "the first source in config order wins")
	assert.Equal(t, "counter", got["only_in_a"].Type)
	assert.Equal(t, "counter", got["described_in_b"].Type, "a source without metadata for a metric must not hide one that has it")
}

func TestSyncer_MetadataSourceFailureCostsOnlyItsOwnMetrics(t *testing.T) {
	provider := &fakeProvider{}
	ok := &fakePromAPI{jobs: []string{"up", "node_load1"}}
	broken := &fakePromAPI{labelErr: errors.New("connection refused")}
	s := newSourcesSyncer(provider, true, metadataSource{name: "broken", api: broken}, metadataSource{name: "ok", api: ok})

	s.RunOnce(context.Background())

	assert.Equal(t, 1, provider.catalogCommits, "the catalog should still be committed from the source that answered")
	assert.ElementsMatch(t, []string{"up", "node_load1"}, keys(catalogByName(provider.catalogItems)))
	assert.Equal(t, 1.0, testutil.ToFloat64(s.metadataSourceFailure.WithLabelValues("broken")))
	assert.Equal(t, 0.0, testutil.ToFloat64(s.metadataSourceFailure.WithLabelValues("ok")))
	assert.Equal(t, 1.0, testutil.ToFloat64(s.syncSuccess), "a partial catalog is still a successful run")
}

func TestSyncer_EveryMetadataSourceFailingFailsTheStep(t *testing.T) {
	provider := &fakeProvider{}
	a := &fakePromAPI{metadataErr: errors.New("a down")}
	b := &fakePromAPI{metadataErr: errors.New("b down")}
	s := newSourcesSyncer(provider, false, metadataSource{name: "a", api: a}, metadataSource{name: "b", api: b})

	s.RunOnce(context.Background())

	assert.Equal(t, 0, provider.catalogCommits)
	assert.Equal(t, 0, provider.summaryCalls, "the summary refresh doesn't run after a failed catalog step")
	assert.Equal(t, 1.0, testutil.ToFloat64(s.syncFailure))
	assert.Equal(t, 0.0, testutil.ToFloat64(s.catalogSummaryMismatch))
}

func TestNewSyncer_ReadsMetadataSourcesWithTheirOwnAuth(t *testing.T) {
	var gotUser, gotPass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"up":[{"type":"gauge","help":"up","unit":""}]}}`))
	}))
	defer srv.Close()

	cfg := *config.DefaultConfig
	src := config.MetadataSourceConfig{URL: srv.URL, HTTPClientConfig: promconfig.DefaultHTTPClientConfig}
	src.HTTPClientConfig.BasicAuth = &promconfig.BasicAuth{Username: "reader", Password: "s3cret"}
	cfg.Inventory.MetadataSources = []config.MetadataSourceConfig{src}

	// The upstream is unreachable on purpose: the catalog step mustn't use it.
	s, err := NewSyncer(&fakeProvider{}, "http://127.0.0.1:1", &cfg, prometheus.NewRegistry())
	require.NoError(t, err)
	require.Len(t, s.metadataSources, 1)

	reported, err := s.fetchCatalog(context.Background(), s.fetchMetadataItems)
	require.NoError(t, err)
	require.Len(t, reported, 1)
	assert.Equal(t, []db.MetricCatalogItem{{Name: "up", Type: "gauge", Help: "up"}}, reported[0].items)
	assert.Equal(t, "reader", gotUser)
	assert.Equal(t, "s3cret", gotPass)
}

func TestNewSyncer_RejectsAMetadataSourceWithoutURL(t *testing.T) {
	cfg := *config.DefaultConfig
	cfg.Inventory.MetadataSources = []config.MetadataSourceConfig{{HTTPClientConfig: promconfig.DefaultHTTPClientConfig}}

	_, err := NewSyncer(&fakeProvider{}, "http://localhost:9090", &cfg, prometheus.NewRegistry())
	assert.ErrorContains(t, err, "inventory.metadata_sources[0]: url is required")
}

func keys(m map[string]db.MetricCatalogItem) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// concurrencyAPI is a source that records how many reads of it, across
// every source sharing it, were ever in flight at once.
type concurrencyAPI struct {
	v1.API

	delay time.Duration

	mu                    sync.Mutex
	inFlight, maxInFlight int
	calls                 int
}

func (c *concurrencyAPI) Metadata(ctx context.Context, _ string, _ string) (map[string][]v1.Metadata, error) {
	c.mu.Lock()
	c.calls++
	c.inFlight++
	c.maxInFlight = max(c.maxInFlight, c.inFlight)
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.inFlight--
		c.mu.Unlock()
	}()

	select {
	case <-time.After(c.delay):
		return map[string][]v1.Metadata{"up": {{Type: "gauge"}}}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestSyncer_MetadataSourceWorkersCapsConcurrentReads(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		shared := &concurrencyAPI{delay: time.Second}
		sources := make([]metadataSource, 7)
		for i := range sources {
			sources[i] = metadataSource{name: string(rune('a' + i)), api: shared}
		}
		s := newSourcesSyncer(&fakeProvider{}, false, sources...)
		s.metadataSourceWorkers = 3

		start := time.Now()
		_, err := s.fetchCatalog(context.Background(), s.fetchMetadataItems)
		require.NoError(t, err)

		assert.Equal(t, 7, shared.calls, "every source should still be read")
		assert.Equal(t, 3, shared.maxInFlight)
		assert.Equal(t, 3*time.Second, time.Since(start), "7 one-second reads, 3 at a time, take 3 rounds")
	})
}

func TestSyncer_MetadataSourceTimeoutFreesAHungSource(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		provider := &fakeProvider{}
		hung := &fakePromAPI{metadataDelay: time.Hour}
		ok := &fakePromAPI{meta: map[string][]v1.Metadata{"up": {{Type: "gauge"}}}}
		// One worker: without the per-source timeout the hung source would
		// hold it until the step's own timeout, and "ok" would never be read.
		s := newSourcesSyncer(provider, false, metadataSource{name: "hung", api: hung}, metadataSource{name: "ok", api: ok})
		s.metadataSourceWorkers = 1
		s.metadataSourceTimeout = 5 * time.Second

		start := time.Now()
		s.RunOnce(context.Background())

		assert.Equal(t, 5*time.Second, time.Since(start))
		assert.Equal(t, 1, provider.catalogCommits)
		assert.ElementsMatch(t, []string{"up"}, keys(catalogByName(provider.catalogItems)))
		assert.Equal(t, 1.0, testutil.ToFloat64(s.metadataSourceFailure.WithLabelValues("hung")))
	})
}

func TestSyncer_SourceTimeoutIsNamedInTheError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSourcesSyncer(&fakeProvider{}, false)
		s.metadataSourceTimeout = 5 * time.Second

		_, err := s.fetchSource(context.Background(), metadataSource{name: "hung", api: &fakePromAPI{metadataDelay: time.Hour}}, s.fetchMetadataItems)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.ErrorContains(t, err, "timed out after 5s (metadata_source_timeout)")
	})
}

func TestSyncer_RecordsWhichSourceReportsEachMetric(t *testing.T) {
	provider := &fakeProvider{}
	first := &fakePromAPI{meta: map[string][]v1.Metadata{
		"up":       {{Type: "gauge"}},
		"req_time": {{Type: "histogram"}},
	}}
	second := &fakePromAPI{meta: map[string][]v1.Metadata{"up": {{Type: "gauge"}}}}
	down := &fakePromAPI{metadataErr: errors.New("down")}
	s := newSourcesSyncer(provider, false,
		metadataSource{name: "source-b", api: first}, metadataSource{name: "source-a", api: second}, metadataSource{name: "down", api: down})

	s.RunOnce(context.Background())

	require.Equal(t, 1, provider.catalogCommits)
	assert.ElementsMatch(t, []string{"up", "req_time_bucket", "req_time_count", "req_time_sum"}, provider.sources["source-b"],
		"a source is recorded against every catalog row it produced, histogram series included")
	assert.Equal(t, []string{"up"}, provider.sources["source-a"])
	assert.NotContains(t, provider.sources, "down", "a source that failed this cycle reported nothing")
	assert.Equal(t, 1, provider.sourcesDeletes, "stale pairs are cleaned up every cycle")
}

func TestSyncer_RecordsNoSourcesWhenReadingTheUpstream(t *testing.T) {
	provider := &fakeProvider{}
	s := newSourcesSyncer(provider, false)
	s.promAPI = &fakePromAPI{meta: map[string][]v1.Metadata{"up": {{Type: "gauge"}}}}

	s.RunOnce(context.Background())

	require.Equal(t, 1, provider.catalogCommits)
	assert.Empty(t, provider.sources)
	assert.Zero(t, provider.sourcesDeletes)
}

func TestSyncer_NamesOnlySourceFillsTheCatalogDescribedSourcesDescribe(t *testing.T) {
	provider := &fakeProvider{}
	nameOnly := true
	// A Thanos-like source: no metadata, but every metric's name.
	names := &fakePromAPI{jobs: []string{"up", "only_in_names"}}
	described := &fakePromAPI{meta: map[string][]v1.Metadata{"up": {{Type: "gauge", Help: "up"}}}}
	s := newSourcesSyncer(provider, false,
		metadataSource{name: "names", api: names, nameOnly: &nameOnly},
		metadataSource{name: "described", api: described})

	s.RunOnce(context.Background())

	require.Equal(t, 1, provider.catalogCommits)
	got := catalogByName(provider.catalogItems)
	assert.Len(t, got, 2)
	assert.Equal(t, "gauge", got["up"].Type, "a described entry wins over a name-only one, whatever the order")
	assert.Equal(t, db.MetricCatalogItem{Name: "only_in_names"}, got["only_in_names"])
}

func TestSyncer_RecordsEachSourcesCardinality(t *testing.T) {
	provider := &fakeProvider{}
	nameOnly := true
	prom := &fakePromAPI{
		meta: map[string][]v1.Metadata{"up": {{Type: "gauge"}}},
		tsdb: v1.TSDBResult{SeriesCountByMetricName: []v1.Stat{{Name: "up", Value: 12}, {Name: "jvm_threads", Value: 40}}},
	}
	thanos := &fakePromAPI{jobs: []string{"up"}}
	broken := &fakePromAPI{
		meta:    map[string][]v1.Metadata{"up": {{Type: "gauge"}}},
		tsdbErr: errors.New("status endpoint disabled"),
	}
	s := newSourcesSyncer(provider, false,
		metadataSource{name: "prom", api: prom},
		metadataSource{name: "thanos", api: thanos, nameOnly: &nameOnly},
		metadataSource{name: "broken", api: broken})
	s.cardinalityLimit = 1000

	s.RunOnce(context.Background())

	require.Equal(t, 1, provider.catalogCommits, "a failed TSDB status doesn't cost the catalog")
	assert.Equal(t, map[string]map[string]int{"prom": {"up": 12, "jvm_threads": 40}}, provider.cardinality,
		"only sources that answered are replaced, so a failure keeps the last numbers")
	assert.Equal(t, 1, prom.tsdbCalls)
	assert.Equal(t, "1", prom.tsdbLimit, "asked with a limit")
	assert.Zero(t, thanos.tsdbCalls, "a name-only source has no TSDB status to give")
	assert.Equal(t, 1, broken.tsdbCalls)
}

func TestSyncer_CardinalityOffReadsNoTSDBStatus(t *testing.T) {
	provider := &fakeProvider{}
	prom := &fakePromAPI{meta: map[string][]v1.Metadata{"up": {{Type: "gauge"}}}}
	s := newSourcesSyncer(provider, false, metadataSource{name: "prom", api: prom})

	s.RunOnce(context.Background())

	assert.Zero(t, prom.tsdbCalls)
	assert.Empty(t, provider.cardinality)
}

func TestSyncer_CardinalityLimitCoversWhatASourceReported(t *testing.T) {
	s := &Syncer{cardinalityLimit: 1000}
	assert.Equal(t, uint64(1000), s.cardinalityLimitFor(48902), "without covering, the fixed top-N")

	s.cardinalityCoverMetadata = true
	assert.Equal(t, uint64(4890), s.cardinalityLimitFor(4890), "every metric the source reported")
	assert.Equal(t, uint64(1000), s.cardinalityLimitFor(300), "never below the configured limit")
	assert.Equal(t, uint64(10000), s.cardinalityLimitFor(48902), "never over what Prometheus accepts")

	s = &Syncer{cardinalityLimit: 50000}
	assert.Equal(t, uint64(10000), s.cardinalityLimitFor(0), "a configured limit over the maximum too")
}
