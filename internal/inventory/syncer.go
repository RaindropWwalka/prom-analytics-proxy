package inventory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/nicolastakashi/prom-analytics-proxy/internal/config"
	"github.com/nicolastakashi/prom-analytics-proxy/internal/db"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	promconfig "github.com/prometheus/common/config"
)

type Syncer struct {
	dbProvider db.Provider
	promAPI    v1.API
	// metadataSources are what the catalog step reads; when empty it reads
	// promAPI, the upstream, like every other step (see catalogSources).
	metadataSources               []metadataSource
	metadataSourceWorkers         int
	metadataSourceTimeout         time.Duration
	metadataSourceRefreshInterval time.Duration
	cardinalityLimit              uint64
	timeWindow                    time.Duration
	metadataLim                   string
	metadataSyncEnabled           bool
	metadataMetricsNameOnly       bool

	runTimeout            time.Duration
	metadataStepTimeout   time.Duration
	summaryStepTimeout    time.Duration
	jobSyncEnabled        bool
	jobIndexTimeout       time.Duration
	jobIndexLabelTimeout  time.Duration
	jobIndexPerJobTimeout time.Duration

	jobIndexWorkers int
	// serviceLabel, when set, has the job index record services too.
	serviceLabel string

	syncDuration           prometheus.Histogram
	syncSuccess            prometheus.Counter
	syncFailure            prometheus.Counter
	catalogSummaryMismatch prometheus.Counter
	jobIndexFailure        prometheus.Counter
	metadataSourceFailure  *prometheus.CounterVec
	catalogSourcesFailure  prometheus.Counter
}

// metadataSource is one API the catalog step reads, named for logs and
// metrics by its URL with any credentials redacted.
type metadataSource struct {
	name string
	api  v1.API
	// nameOnly, when set, overrides the syncer-wide choice of endpoint.
	nameOnly *bool
}

// stepOverrunAllowance is what a cycle budget has to carry, per enabled
// step, on top of the step budgets themselves. A step reaching its own
// deadline still has to notice the cancellation and unwind, and cancelling
// an in-flight Prometheus or Postgres round trip costs a round trip of its
// own, so a step's wall-clock duration runs over the budget it was given by
// a little. Cover that here and the overrun is spent out of the cycle's own
// allowance; leave it uncovered and it is spent out of the next step's
// window, which is the truncation independent step budgets exist to prevent.
// An allowance rather than a measurement, and deliberately well above what
// abandoning a round trip should cost: over-provisioning it only raises the
// floor RunTimeout has to clear, where under-provisioning it puts the
// truncation back.
const stepOverrunAllowance = 10 * time.Second

// SettleConfig fixes cfg's inventory timeouts at the values a cycle will
// actually run under, and is what makes run_timeout mean the same thing to
// every reader: startup calls it once, before anything reads the config, so
// whatever declares this job's cycle budget can read the config rather than
// asking separately. Idempotent, which is what lets NewSyncer settle again
// for callers that never went through startup.
func SettleConfig(cfg *config.Config) {
	cfg.Inventory.JobIndexTimeout = settledJobIndexTimeout(cfg.Inventory)
	cfg.Inventory.RunTimeout = settledRunTimeout(cfg.Inventory)
}

// A cycle's enabled steps are independent, non-overlapping budgets, so its
// worst case is all of them in full plus their overrun allowance, and
// RunTimeout has to cover that floor; the same holds one level down, where
// JobIndexTimeout has to cover its own label fetch plus at least one job's
// query. Where a container is too small for what it must hold, the two
// settled* functions below widen it to its floor and warn naming the value
// to fix, rather than refusing to start: the operator asked for those step
// budgets, and running with them is closer to their intent than not running
// at all.
// settledRunTimeout reads JobIndexTimeout as given, so a caller wanting both
// settled has to settle that one first - as SettleConfig does.
func settledRunTimeout(cfg config.InventoryConfig) time.Duration {
	// The summary step is never gated by a flag, so it always counts.
	stepSum, steps := cfg.SummaryStepTimeout, 1
	if cfg.MetadataSyncEnabled {
		stepSum += cfg.MetadataStepTimeout
		steps++
	}
	if cfg.JobSyncEnabled {
		stepSum += cfg.JobIndexTimeout
		steps++
	}
	allowance := time.Duration(steps) * stepOverrunAllowance
	floor := stepSum + allowance
	if floor <= cfg.RunTimeout {
		return cfg.RunTimeout
	}
	slog.Warn("inventory: run_timeout is shorter than the steps one cycle must run plus their overrun allowance; raising it - set it to at least this floor in your config to silence this",
		"configured", cfg.RunTimeout,
		"using", floor,
		"step_sum", stepSum,
		"overrun_allowance", allowance,
		"metadata_step_timeout", cfg.MetadataStepTimeout,
		"metadata_sync_enabled", cfg.MetadataSyncEnabled,
		"summary_step_timeout", cfg.SummaryStepTimeout,
		"job_index_timeout", cfg.JobIndexTimeout,
		"job_sync_enabled", cfg.JobSyncEnabled)
	return floor
}

// settledJobIndexTimeout's widening feeds the step sum above, so it can
// widen the cycle that has to contain it.
func settledJobIndexTimeout(cfg config.InventoryConfig) time.Duration {
	floor := cfg.JobIndexLabelTimeout + cfg.JobIndexPerJobTimeout
	if !cfg.JobSyncEnabled || cfg.JobIndexTimeout >= floor {
		return cfg.JobIndexTimeout
	}
	slog.Warn("inventory: job_index_timeout is too short to fetch job labels and process a single job; raising it - set it to at least this sum in your config to silence this",
		"configured", cfg.JobIndexTimeout,
		"using", floor,
		"job_index_label_timeout", cfg.JobIndexLabelTimeout,
		"job_index_per_job_timeout", cfg.JobIndexPerJobTimeout)
	return floor
}

// NewSyncer builds a Syncer whose cycles run under settled timeouts (see
// above) and rejects the config values nothing could settle - a non-positive
// interval, no workers to fan out to. Settling is what lets RunOnce nest
// every step under one context bounded by RunTimeout while each step still
// gets its own window in full - see docs/jobs.md, and
// https://github.com/nicolastakashi/prom-analytics-proxy/issues/572 for the
// starvation this generalizes away.
func NewSyncer(dbp db.Provider, upstream string, cfg *config.Config, reg prometheus.Registerer) (*Syncer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	client, err := api.NewClient(api.Config{Address: upstream})
	if err != nil {
		return nil, err
	}

	// Range checks live with the schema (see config.InventoryConfig.Validate);
	// startup runs them before anything acts on the config, and calling them
	// again here is what holds for a caller that never went through startup.
	if err := cfg.Inventory.Validate(); err != nil {
		return nil, err
	}

	// A copy, not cfg itself: a constructor that rewrote its caller's config
	// would surprise anyone sharing one. Every field below reads from this
	// copy rather than cfg.Inventory, so a field that gains a settling rule
	// later gets it whether or not whoever adds it notices the two sources.
	inv := cfg.Inventory
	inv.JobIndexTimeout = settledJobIndexTimeout(inv)
	inv.RunTimeout = settledRunTimeout(inv)

	// What cycles actually run on, where the startup config dump prints what
	// was configured - the two differ whenever settling had to widen one.
	// Only the values settling can change, so nothing here reads as a second
	// source for the rest.
	slog.Info("inventory: settled cycle budget",
		"run_timeout", inv.RunTimeout,
		"job_index_timeout", inv.JobIndexTimeout)

	sources := make([]metadataSource, 0, len(inv.MetadataSources))
	for i, src := range inv.MetadataSources {
		if err := src.Validate(); err != nil {
			return nil, fmt.Errorf("inventory.metadata_sources[%d]: %w", i, err)
		}
		name := src.DisplayName()
		rt, err := promconfig.NewRoundTripperFromConfig(src.HTTPClientConfig, "prom-analytics-proxy-inventory")
		if err != nil {
			return nil, fmt.Errorf("inventory.metadata_sources[%d] (%s): %w", i, name, err)
		}
		c, err := api.NewClient(api.Config{Address: src.URL, RoundTripper: rt})
		if err != nil {
			return nil, fmt.Errorf("inventory.metadata_sources[%d] (%s): %w", i, name, err)
		}
		sources = append(sources, metadataSource{name: name, api: v1.NewAPI(c), nameOnly: src.MetricsNameOnly})
	}

	lim := ""
	if cfg.MetadataLimit > 0 {
		lim = strconv.FormatUint(cfg.MetadataLimit, 10)
	}
	s := &Syncer{
		dbProvider:                    dbp,
		promAPI:                       v1.NewAPI(client),
		metadataSources:               sources,
		metadataSourceWorkers:         inv.MetadataSourceWorkers,
		metadataSourceTimeout:         inv.MetadataSourceTimeout,
		metadataSourceRefreshInterval: inv.MetadataSourceRefreshInterval,
		cardinalityLimit:              inv.CardinalityLimit,
		timeWindow:                    inv.TimeWindow,
		metadataLim:                   lim,
		metadataSyncEnabled:           inv.MetadataSyncEnabled,
		metadataMetricsNameOnly:       inv.MetadataMetricsNameOnly,
		runTimeout:                    inv.RunTimeout,
		metadataStepTimeout:           inv.MetadataStepTimeout,
		summaryStepTimeout:            inv.SummaryStepTimeout,
		jobSyncEnabled:                inv.JobSyncEnabled,
		jobIndexTimeout:               inv.JobIndexTimeout,
		jobIndexLabelTimeout:          inv.JobIndexLabelTimeout,
		jobIndexPerJobTimeout:         inv.JobIndexPerJobTimeout,
		jobIndexWorkers:               inv.JobIndexWorkers,
		serviceLabel:                  inv.ServiceLabel,
	}

	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	s.syncDuration = promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
		Name:    "inventory_sync_duration_seconds",
		Help:    "Duration of inventory sync runs in seconds",
		Buckets: prometheus.DefBuckets,
	})
	s.syncSuccess = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "inventory_sync_success_total",
		Help: "Total number of successful inventory sync runs",
	})
	s.syncFailure = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "inventory_sync_failure_total",
		Help: "Total number of failed inventory sync runs",
	})
	s.catalogSummaryMismatch = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "inventory_catalog_summary_mismatch_total",
		Help: "Total number of runs where the metadata catalog was committed but the usage-summary refresh failed, stranding the affected metrics at placeholder values",
	})
	s.jobIndexFailure = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "inventory_job_index_failure_total",
		Help: "Total number of runs where the job-index step failed, leaving the index stale for the jobs it could not reach while the run itself still counted as a success",
	})
	s.metadataSourceFailure = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Name: "inventory_metadata_source_failure_total",
		Help: "Total number of catalog steps in which a metadata source could not be read; the catalog is still committed from the sources that answered unless every one of them failed",
	}, []string{"source"})
	// Exported at zero from the start, so a source that has never failed
	// still has a series to alert on.
	for _, src := range sources {
		s.metadataSourceFailure.WithLabelValues(src.name)
	}
	s.catalogSourcesFailure = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "inventory_catalog_sources_failure_total",
		Help: "Total number of failed writes recording which metadata source reports which metric; the catalog itself was already committed",
	})

	return s, nil
}

// RunOnce runs exactly one sync cycle, bounded by whatever ctx it's given -
// scheduling and leadership are entirely the caller's concern; RunOnce
// itself doesn't loop or know who's calling it.
//
// cycleCtx is the single deadline every step below derives from, bounded
// by runTimeout. Each step still nests its own context.WithTimeout inside
// it and gets that window in full; NewSyncer's validation of runTimeout
// against the step sum is what makes that safe.
func (s *Syncer) RunOnce(ctx context.Context) {
	start := time.Now()
	cycleCtx, cancel := context.WithTimeout(ctx, s.runTimeout)
	defer cancel()

	catalogCommitted, err := s.syncCatalogAndSummary(cycleCtx)
	if err != nil {
		s.syncFailure.Inc()
		if catalogCommitted {
			// The metadata catalog step already committed (or was skipped)
			// this run, so any newly-catalogued metrics now sit at their
			// default placeholder summary values (is_unused=true, zero
			// counts) until a future run successfully refreshes them.
			// Surface this as its own signal so it isn't buried in the
			// generic failure counter.
			s.catalogSummaryMismatch.Inc()
			slog.Warn("inventory: catalog committed but usage-summary refresh failed this run; affected metrics remain at placeholder values until the next scheduled sync",
				"err", err)
		}
		s.syncDuration.Observe(time.Since(start).Seconds())
		return
	}

	if s.jobSyncEnabled {
		if err := s.syncJobIndex(cycleCtx, db.TimeRangeUntilNow(s.timeWindow)); err != nil {
			// The run still counts as a success: the catalog and the summary
			// both landed, and this step's outcome is separable from theirs
			// (see docs/jobs.md). Counted on its own so an index left stale
			// for most of the jobs is queryable, rather than a log line
			// nothing alerts on.
			s.jobIndexFailure.Inc()
			slog.Error("inventory: job index", "err", err)
		}
	}

	slog.Info("inventory: sync complete")
	s.syncSuccess.Inc()
	s.syncDuration.Observe(time.Since(start).Seconds())
}

// syncCatalogAndSummary runs the metadata-catalog sync followed by the
// usage-summary refresh. It reports whether the catalog step committed (or
// was intentionally skipped) so the caller can tell an ordinary failure
// apart from the partial-failure case where the catalog moved forward but
// the summary refresh didn't.
func (s *Syncer) syncCatalogAndSummary(cycleCtx context.Context) (catalogCommitted bool, err error) {
	if s.metadataSyncEnabled {
		if err := s.syncMetadataCatalog(cycleCtx); err != nil {
			return false, err
		}
	} else {
		slog.Info("inventory: metadata sync disabled, skipping catalog population")
	}

	sumCtx, cancelSum := context.WithTimeout(cycleCtx, s.summaryStepTimeout)
	defer cancelSum()
	if err := s.dbProvider.RefreshMetricsUsageSummary(sumCtx, db.TimeRangeUntilNow(s.timeWindow)); err != nil {
		slog.Error("inventory: refresh summary", "err", err)
		return true, err
	}
	return true, nil
}

// syncMetadataCatalog reads every catalog source - the upstream, or the
// configured metadata_sources - and commits what they returned as one
// catalog. A source that fails costs only its own metrics: the step fails
// only when no source answered, since committing nothing would gain nothing
// over committing what the reachable ones know.
func (s *Syncer) syncMetadataCatalog(ctx context.Context) error {
	metaCtx, cancelMeta := context.WithTimeout(ctx, s.metadataStepTimeout)
	defer cancelMeta()

	fetch := s.fetchMetadataItems
	if s.metadataMetricsNameOnly {
		fetch = s.fetchMetricNameItems
	}
	reported, err := s.fetchCatalog(metaCtx, fetch)
	if err != nil {
		return err
	}
	if err := s.dbProvider.UpsertMetricsCatalog(metaCtx, mergeCatalogItems(reported)); err != nil {
		slog.Error("inventory: upsert catalog", "err", err)
		return err
	}
	if len(s.metadataSources) > 0 {
		s.recordCatalogSources(metaCtx, reported)
	}
	return nil
}

// sourceCatalog is what one source returned in a cycle.
type sourceCatalog struct {
	source string
	items  []db.MetricCatalogItem
	// series is the source's head series per metric (its top
	// cardinalityLimit), or nil when it wasn't read this cycle.
	series map[string]int
}

// recordCatalogSources keeps track of which source reports which metric.
// It runs after the catalog itself has committed, so its failures are
// counted and logged but don't fail the step: the catalog is the step's
// product, and which sources report a metric is extra detail on it.
func (s *Syncer) recordCatalogSources(ctx context.Context, reported []sourceCatalog) {
	now := time.Now()
	fail := func(msg string, args ...any) {
		slog.Error(msg, args...)
		if s.catalogSourcesFailure != nil {
			s.catalogSourcesFailure.Inc()
		}
	}
	for _, r := range reported {
		names := make([]string, len(r.items))
		for i, it := range r.items {
			names[i] = it.Name
		}
		if err := s.dbProvider.UpsertMetricsCatalogSources(ctx, r.source, names, now.Add(-s.metadataSourceRefreshInterval)); err != nil {
			fail("inventory: record catalog sources", "source", r.source, "err", err)
		}
	}
	for _, r := range reported {
		if r.series == nil {
			continue
		}
		if err := s.dbProvider.ReplaceMetricsCardinality(ctx, r.source, r.series); err != nil {
			fail("inventory: record cardinality", "source", r.source, "err", err)
		}
	}
	// A pair unconfirmed for a whole time_window is a metric its source no
	// longer reports - the same horizon after which the summary stops
	// refreshing a catalog row.
	if _, err := s.dbProvider.DeleteMetricsCatalogSourcesBefore(ctx, now.Add(-s.timeWindow)); err != nil {
		fail("inventory: delete stale catalog sources", "err", err)
	}
}

// catalogSources is the upstream unless metadata_sources were configured.
func (s *Syncer) catalogSources() []metadataSource {
	if len(s.metadataSources) > 0 {
		return s.metadataSources
	}
	return []metadataSource{{name: "upstream", api: s.promAPI}}
}

// fetchCatalog reads the catalog sources, at most metadataSourceWorkers at
// a time and in config order, all under the one metadata step budget, and
// returns what each source that answered reported, in config order.
func (s *Syncer) fetchCatalog(ctx context.Context, fetch func(context.Context, v1.API) ([]db.MetricCatalogItem, error)) ([]sourceCatalog, error) {
	sources := s.catalogSources()
	results := make([][]db.MetricCatalogItem, len(sources))
	errs := make([]error, len(sources))

	workers := s.metadataSourceWorkers
	if workers <= 0 || workers > len(sources) {
		workers = len(sources)
	}
	series := make([]map[string]int, len(sources))
	next := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for i := range next {
				results[i], errs[i] = s.fetchSource(ctx, sources[i], fetch)
				if errs[i] == nil && s.readsCardinality(sources[i]) {
					series[i] = s.fetchCardinality(ctx, sources[i])
				}
			}
		})
	}
	for i := range sources {
		next <- i
	}
	close(next)
	wg.Wait()

	var failed []error
	for i, err := range errs {
		if err != nil {
			failed = append(failed, fmt.Errorf("%s: %w", sources[i].name, err))
		}
	}
	if len(failed) == len(sources) {
		return nil, errors.Join(failed...)
	}
	if len(failed) > 0 {
		slog.Warn("inventory: some metadata sources failed; committing the catalog from the rest",
			"failed", len(failed), "sources", len(sources))
	}
	reported := make([]sourceCatalog, 0, len(sources)-len(failed))
	for i, src := range sources {
		if errs[i] == nil {
			reported = append(reported, sourceCatalog{source: src.name, items: results[i], series: series[i]})
		}
	}
	return reported, nil
}

// fetchSource reads one source under its own timeout, when one is set, and
// logs the outcome as soon as it has one: with tens of sources a step can
// run for many minutes, and a failure reported only at its end is a failure
// nobody sees in time.
func (s *Syncer) fetchSource(ctx context.Context, src metadataSource, fetch func(context.Context, v1.API) ([]db.MetricCatalogItem, error)) ([]db.MetricCatalogItem, error) {
	srcCtx := ctx
	if s.metadataSourceTimeout > 0 {
		var cancel context.CancelFunc
		srcCtx, cancel = context.WithTimeout(ctx, s.metadataSourceTimeout)
		defer cancel()
	}
	if src.nameOnly != nil {
		fetch = s.fetchMetadataItems
		if *src.nameOnly {
			fetch = s.fetchMetricNameItems
		}
	}
	start := time.Now()
	items, err := fetch(srcCtx, src.api)
	took := time.Since(start).Round(time.Millisecond)
	if err != nil {
		// Its own deadline, not the step's: the step's would fail every
		// source left, and say so itself.
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			err = fmt.Errorf("timed out after %s (metadata_source_timeout): %w", s.metadataSourceTimeout, err)
		}
		slog.Error("inventory: metadata source failed", "source", src.name, "took", took, "err", err)
		if s.metadataSourceFailure != nil {
			s.metadataSourceFailure.WithLabelValues(src.name).Inc()
		}
		return nil, err
	}
	slog.Info("inventory: metadata source read", "source", src.name, "metrics", len(items), "took", took)
	return items, nil
}

// readsCardinality tells whether src is asked for its TSDB status: only
// configured metadata sources (the upstream may be a Thanos Querier) that
// read metadata - a name-only source is one that has no TSDB of its own.
func (s *Syncer) readsCardinality(src metadataSource) bool {
	if s.cardinalityLimit == 0 || len(s.metadataSources) == 0 {
		return false
	}
	nameOnly := s.metadataMetricsNameOnly
	if src.nameOnly != nil {
		nameOnly = *src.nameOnly
	}
	return !nameOnly
}

// fetchCardinality reads src's head series per metric, its top
// cardinalityLimit. It returns nil on failure, logged: the catalog doesn't
// depend on it, and the last cycle's numbers stay until a read succeeds.
func (s *Syncer) fetchCardinality(ctx context.Context, src metadataSource) map[string]int {
	if s.metadataSourceTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.metadataSourceTimeout)
		defer cancel()
	}
	status, err := src.api.TSDB(ctx, v1.WithLimit(s.cardinalityLimit))
	if err != nil {
		slog.Warn("inventory: read cardinality", "source", src.name, "err", err)
		return nil
	}
	series := make(map[string]int, len(status.SeriesCountByMetricName))
	for _, stat := range status.SeriesCountByMetricName {
		series[stat.Name] = int(stat.Value)
	}
	return series
}

// mergeCatalogItems keeps one item per metric name across sources. The
// first source, in config order, to know a metric's type is the one whose
// metadata is kept, so a name-only entry never blanks out a described one.
func mergeCatalogItems(reported []sourceCatalog) []db.MetricCatalogItem {
	if len(reported) == 1 {
		return reported[0].items
	}
	index := make(map[string]int)
	var merged []db.MetricCatalogItem
	for _, r := range reported {
		for _, it := range r.items {
			i, seen := index[it.Name]
			if !seen {
				index[it.Name] = len(merged)
				merged = append(merged, it)
				continue
			}
			if merged[i].Type == "" && it.Type != "" {
				merged[i] = it
			}
		}
	}
	return merged
}

func (s *Syncer) fetchMetadataItems(ctx context.Context, promAPI v1.API) ([]db.MetricCatalogItem, error) {
	meta, err := promAPI.Metadata(ctx, "", s.metadataLim)
	if err != nil {
		return nil, err
	}

	items := make([]db.MetricCatalogItem, 0, len(meta)*2)
	for name, infos := range meta {
		if len(infos) == 0 {
			continue
		}
		info := infos[0]
		metricType := string(info.Type)
		switch metricType {
		case "histogram":
			items = append(items,
				db.MetricCatalogItem{Name: name + "_bucket", Type: "histogram_bucket", Help: info.Help + " (histogram buckets)", Unit: info.Unit},
				db.MetricCatalogItem{Name: name + "_count", Type: "histogram_count", Help: info.Help + " (histogram count)", Unit: ""},
				db.MetricCatalogItem{Name: name + "_sum", Type: "histogram_sum", Help: info.Help + " (histogram sum)", Unit: info.Unit},
			)
		case "summary":
			items = append(items,
				db.MetricCatalogItem{Name: name, Type: metricType, Help: info.Help, Unit: info.Unit},
				db.MetricCatalogItem{Name: name + "_count", Type: "summary_count", Help: info.Help + " (summary count)", Unit: ""},
				db.MetricCatalogItem{Name: name + "_sum", Type: "summary_sum", Help: info.Help + " (summary sum)", Unit: info.Unit},
			)
		default:
			items = append(items, db.MetricCatalogItem{Name: name, Type: metricType, Help: info.Help, Unit: info.Unit})
		}
	}
	return items, nil
}

func (s *Syncer) fetchMetricNameItems(ctx context.Context, promAPI v1.API) ([]db.MetricCatalogItem, error) {
	now := time.Now()
	labelValues, _, _, err := promAPI.LabelValues(ctx, "__name__", nil, now.Add(-s.timeWindow), now)
	if err != nil {
		return nil, err
	}

	items := make([]db.MetricCatalogItem, 0, len(labelValues))
	for _, labelValue := range labelValues {
		if !labelValue.IsValid() {
			continue
		}
		items = append(items, db.MetricCatalogItem{Name: string(labelValue)})
	}
	return items, nil
}
