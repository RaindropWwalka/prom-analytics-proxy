package inventory

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
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
		dbProvider:              provider,
		metadataSources:         sources,
		timeWindow:              time.Hour,
		metadataSyncEnabled:     true,
		metadataMetricsNameOnly: nameOnly,
		runTimeout:              time.Minute,
		metadataStepTimeout:     30 * time.Second,
		summaryStepTimeout:      30 * time.Second,
		syncDuration:            newHistogram(),
		syncSuccess:             newCounter(),
		syncFailure:             newCounter(),
		catalogSummaryMismatch:  newCounter(),
		metadataSourceFailure:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_source_failure"}, []string{"source"}),
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
	s := newSourcesSyncer(provider, false, metadataSource{"a", a}, metadataSource{"b", b})

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
	s := newSourcesSyncer(provider, true, metadataSource{"broken", broken}, metadataSource{"ok", ok})

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
	s := newSourcesSyncer(provider, false, metadataSource{"a", a}, metadataSource{"b", b})

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

	items, err := s.fetchCatalog(context.Background(), s.fetchMetadataItems)
	require.NoError(t, err)
	assert.Equal(t, []db.MetricCatalogItem{{Name: "up", Type: "gauge", Help: "up"}}, items)
	assert.Equal(t, "reader", gotUser)
	assert.Equal(t, "s3cret", gotPass)
}

func TestNewSyncer_RejectsAMetadataSourceWithoutURL(t *testing.T) {
	cfg := *config.DefaultConfig
	cfg.Inventory.MetadataSources = []config.MetadataSourceConfig{{HTTPClientConfig: promconfig.DefaultHTTPClientConfig}}

	_, err := NewSyncer(&fakeProvider{}, "http://localhost:9090", &cfg, prometheus.NewRegistry())
	assert.ErrorContains(t, err, "inventory.metadata_sources[0]: url is required")
}

func TestRedactURL_HidesPasswords(t *testing.T) {
	assert.Equal(t, "https://user:xxxxx@prom.example.com", redactURL("https://user:pass@prom.example.com"))
	assert.Equal(t, "http://prom:9090", redactURL("http://prom:9090"))
}

func keys(m map[string]db.MetricCatalogItem) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
