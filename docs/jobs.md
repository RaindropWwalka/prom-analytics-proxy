# Background jobs

`internal/inventory` and `internal/retention` are the two periodic jobs `prom-analytics-proxy` runs against its own database: the inventory syncer builds and refreshes the metrics catalog and usage data the rest of the application reads, and the retention worker deletes old query-log rows. Both expose the same shape — a `RunOnce(ctx)` that runs exactly one cycle, bounded by whatever `ctx` it's given, and never loops or knows who's calling it — so `cmd/api` can drive them identically regardless of database provider. This doc covers what each job does and the constraints it operates under; *who* gets to run them, and how a stuck cycle gets recovered, is [docs/leader-election.md](leader-election.md)'s concern — see [Running these jobs](#running-these-jobs) for where the two connect.

## Inventory syncer

`internal/inventory.Syncer` maintains two tables: the metrics catalog (name, type, help, unit — what metrics exist) and, optionally, a per-Prometheus-job index (which metrics each job produces), used to answer job-scoped "which of this job's metrics are unused" queries. One call to `RunOnce` runs up to three independent steps, strictly in this order:

1. **Metadata catalog sync** — populates the catalog from Prometheus. Skipped entirely if `metadata_sync_enabled` is `false` (the OTLP ingester populates the catalog instead in that mode — see [docs/ingester.md](ingester.md)). Two mutually exclusive sources, selected by `metadata_metrics_name_only`: the `/api/v1/metadata` endpoint by default (full type/help/unit; expands histograms and summaries into their `_bucket`/`_count`/`_sum` series; capped by the top-level `metadata_limit`), or `/api/v1/label/__name__/values` when set (names only — lighter, and less prone to hitting Prometheus's own metadata limits, at the cost of a less complete catalog). Read from the upstream by default, or from every entry in `metadata_sources` when that is set (see [Metadata sources](#metadata-sources)). Bounded by `metadata_step_timeout`.
2. **Usage-summary refresh** — always runs, never gated by a flag. Bounded by its own `summary_step_timeout`.
3. **Job index sync** (`internal/inventory/job_index.go`) — only if `job_sync_enabled`, and only after steps 1–2 both succeed. Fetches the set of Prometheus job label values, then fans out across `job_index_workers` concurrent goroutines, each pulling jobs off a shared channel and querying-then-upserting one job's metric names at a time. Bounded as a *whole* by `job_index_timeout`; the label fetch (`job_index_label_timeout`) and each job's own turn — its query and its commit both (`job_index_per_job_timeout`) — are sub-budgets nested inside it, not independent of it, so one job stuck on a slow query or a database lock costs its own budget rather than the whole step's. An empty label result or a 404 is treated as "no job labels exist" and logged, not an error; the step overall fails only if more than half of the individual jobs failed.

Steps 1 and 2 are one unit for error reporting: `RunOnce` stops there and reports which of two failure shapes happened — an ordinary failure (nothing committed) versus catalog-committed-but-summary-failed, counted separately (`inventory_catalog_summary_mismatch_total`) because it strands newly-catalogued metrics at placeholder usage values until the next successful run. Step 3's failures don't affect steps 1–2's outcome or the run's own success/failure counters — they get a counter of their own (`inventory_job_index_failure_total`), so an index left stale is visible without a partial job index costing a committed catalog and a refreshed summary their success — and step 3 never runs at all if step 1 or 2 failed.

### Time budget

Every step gets its own window sized by its own timeout, rather than a share of whatever an earlier step left behind. All three derive from one shared context — `RunOnce` opens a single `cycleCtx` bounded by `run_timeout`, and every step's own `context.WithTimeout` nests inside it — and `NewSyncer` is what makes that safe, settling at construction on a `run_timeout` at least as large as a floor it computes: the sum of every *enabled* step's own timeout, plus a fixed overrun allowance per enabled step. A configured value smaller than that floor is raised to it, with a warning naming the value to fix, rather than refusing to start. Running the steps an operator asked for is closer to their intent than not running at all. A step that finishes early leaves its unused window behind; it never lends to, or borrows from, another step.

The allowance is what keeps the step budgets honest in wall-clock terms rather than only as configured values. A step reaching its own deadline still has to notice the cancellation and unwind, and cancelling an in-flight Prometheus or Postgres round trip costs a round trip of its own, so a step's real duration runs a little over the budget it was given. Covered by the allowance, that overrun is spent out of the cycle's own margin; left uncovered, it comes out of the next step's window — which is exactly the truncation independent step budgets exist to prevent, and the last step in the order absorbs all of it. The shipped defaults sit exactly on the floor: `30s + 30s + 210s` of steps, plus `3 × 10s` of allowance, is the `300s` `run_timeout` default.

This is why `job_index_timeout` exists as its own flag distinct from `job_index_label_timeout`/`job_index_per_job_timeout`: those two bound individual calls, not the step's total duration, which otherwise scales with however many Prometheus jobs are discovered at run time — something no per-call timeout alone can cap. Without a step-level cap of its own, job index would have no accurate contribution to declare toward the overall budget, the same problem already fixed for the summary step (see `TestSyncer_SlowMetadataStepDoesNotPermanentlyStarveSummaryRefresh`).

Settled this way, `run_timeout` is a true bound on one whole cycle rather than on part of one — which is what makes it usable as this job's declared cycle budget, with no separate computation needed anywhere else. `inventory.SettleConfig` is what makes that true for every reader rather than only for the syncer: startup calls it once, before anything reads the config, so a widened `run_timeout` is simply the value in the config from then on, and nothing has to ask separately what a cycle really runs under — see [Reporting job cycles](leader-election.md#reporting-job-cycles) for who reads it, and [Running these jobs](#running-these-jobs).

### Config

| Field (`inventory.*`) | Flag | Default | Governs |
|---|---|---|---|
| `enabled` | `-inventory-enabled` | `true` | Whether this job runs at all |
| `metadata_sync_enabled` | `-inventory-metadata-sync-enabled` | `true` | Whether step 1 runs (and counts toward `run_timeout`'s required minimum) |
| `metadata_metrics_name_only` | — (YAML only) | `false` | Which source step 1 uses |
| `metadata_sources` | — (YAML only) | none | Where step 1 reads from instead of the upstream — see [Metadata sources](#metadata-sources) |
| `metadata_source_workers` | — (YAML only) | `4` | How many `metadata_sources` step 1 reads at once |
| `metadata_source_refresh_interval` | — (YAML only) | `24h` | How long a record of a source reporting a metric stands before step 1 rewrites it — must be shorter than `time_window` |
| `metadata_source_timeout` | — (YAML only) | `1m` | Bounds one source's read, nested inside `metadata_step_timeout`; `0s` leaves each read bounded by `metadata_step_timeout` alone |
| `sync_interval` | `-inventory-sync-interval` | `10m` | Time between cycles (jittered up to 20%, so replicas restarting together don't all sync in lockstep) |
| `time_window` | `-inventory-time-window` | `720h` (30d) | Lookback window for the usage-summary refresh and job-index queries |
| `run_timeout` | `-inventory-run-timeout` | `300s` | The whole cycle's real deadline — must be at least the sum of every enabled step's own timeout below plus `10s` of overrun allowance per enabled step; a smaller value is raised to that floor at startup, with a warning naming it |
| `metadata_step_timeout` | `-inventory-metadata-timeout` | `30s` | Bounds step 1 alone |
| `summary_step_timeout` | `-inventory-summary-timeout` | `30s` | Bounds step 2 alone |
| `job_sync_enabled` | — (YAML only) | `false` | Whether step 3 runs (and counts toward `run_timeout`'s required minimum) |
| `job_index_timeout` | `-inventory-job-index-timeout` | `210s` | Bounds step 3 as a whole — must be at least `job_index_label_timeout + job_index_per_job_timeout`, or the step could never index a single job; a smaller value is raised to that sum at startup, with a warning |
| `job_index_label_timeout` | `-inventory-job-index-label-timeout` | `30s` | Bounds the job-label fetch, nested inside `job_index_timeout` |
| `job_index_per_job_timeout` | `-inventory-job-index-per-job-timeout` | `30s` | Bounds one job's whole turn — its query and its commit — nested inside `job_index_timeout` |
| `job_index_workers` | `-inventory-job-index-workers` | `10` | Concurrency of step 3's per-job fan-out |

### Metadata sources

Step 1 reads the upstream by default. That doesn't work when the upstream can't serve metadata: Thanos Receive, for one, keeps none, so a Querier in front of it answers `/api/v1/metadata` with an empty result, while the Prometheus servers remote-writing into it still could. `metadata_sources` points step 1 at those servers instead; queries, step 2 and step 3 keep going through the upstream.

```yaml
inventory:
  metadata_sources:
  - name: local
    url: http://prometheus.monitoring.svc:9090
  - name: other-cluster
    url: https://prometheus.other-cluster.example.com
    basic_auth:
      username: reader
      password_file: /etc/secrets/prometheus-password
```

Each entry is a `url` plus the HTTP client block Prometheus itself uses (`basic_auth`, `authorization`, `oauth2`, `tls_config`, `proxy_url`, ...). Sources are read in config order, `metadata_source_workers` at a time — so a long list doesn't land every response in memory at once — all within the one `metadata_step_timeout`; `metadata_source_timeout`, when set, bounds each read, so a source that hangs gives up its worker instead of holding it until the step times out and starving the sources queued behind it. Size `metadata_step_timeout` for the sources one worker takes in turn, not for a single read. The results are merged — for a metric several sources know, the first source in config order that has its type wins, so a source without metadata for it never blanks out one that has it — and the merged catalog is committed in one upsert. A source that fails costs only its own metrics: the step still commits what the others returned, and fails only if none answered. Every source's outcome is logged as soon as it has one — `inventory: metadata source read` with how many metrics it returned and how long it took, or `inventory: metadata source failed` with the error, a source timeout named as such — rather than once the step ends, which with tens of sources can be many minutes later; each failure is also counted in `inventory_metadata_source_failure_total`.

Step 1 also records which source reports which metric, in `metrics_catalog_sources` — one row per (metric, source), under the source's `name` (its URL, password redacted, when unset; names must be unique). The API returns them as `sources` on every `/api/v1/seriesMetadata` row, and lists every source that reports anything at `/api/v1/sources`. With tens of sources each reporting tens of thousands of metrics, rewriting every row every cycle would dwarf the catalog write itself, so a row is only written when it's new or when it was last confirmed more than `metadata_source_refresh_interval` ago; a row left unconfirmed for a whole `time_window` is a metric its source stopped reporting, and is deleted. These writes happen after the catalog has committed, within the same `metadata_step_timeout`; a failure is logged and counted in `inventory_catalog_sources_failure_total` but doesn't fail the step. Nothing is recorded when step 1 reads the upstream.

### Metrics

- `inventory_sync_duration_seconds` (histogram) — wall time of one `RunOnce` call, observed once regardless of outcome.
- `inventory_sync_success_total` / `inventory_sync_failure_total` (counters) — steps 1–2's outcome only; step 3 isn't reflected here.
- `inventory_catalog_summary_mismatch_total` (counter) — the catalog-committed-but-summary-failed partial-failure case specifically.
- `inventory_job_index_failure_total` (counter) — step 3 failed, so the index is stale for the jobs it couldn't reach, while the run itself still counted as a success. Alert on this rather than on the success counter, which by design says nothing about step 3.
- `inventory_metadata_source_failure_total` (counter, label `source`) — a `metadata_sources` entry couldn't be read in a cycle's step 1. The catalog was still committed from the others unless every source failed, in which case the run is counted as a failure too. The label is the source's `name`.
- `inventory_catalog_sources_failure_total` (counter) — recording which source reports which metric failed after the catalog committed; the `sources` the API shows are stale until a later cycle succeeds.

## Retention worker

`internal/retention.Worker` deletes query-log rows older than `queries_max_age`. One call to `RunOnce` is a single step: compute the cutoff (`now - queries_max_age`), call `dbProvider.DeleteQueriesBefore`, bounded by `run_timeout`. No further steps, so unlike the inventory syncer there's nothing to validate or sum — `run_timeout` is already, directly, the whole cycle's true bound. If `queries_max_age` is zero or negative the run is a no-op (defensive; `NewWorker` already rejects a non-positive value at construction, so this only guards against a value changed after startup).

### Config

| Field (`retention.*`) | Flag | Default | Governs |
|---|---|---|---|
| `enabled` | `-retention-enabled` | `false` | Whether this job runs at all |
| `interval` | `-retention-interval` | `1h` | Time between cycles (jittered up to 20%) |
| `run_timeout` | `-retention-run-timeout` | `5m` | Bounds the single delete step — the whole cycle's true bound |
| `queries_max_age` | `-retention-queries-max-age` | `720h` (30d) | Cutoff age — rows older than this are deleted |

### Metrics

- `retention_run_duration_seconds` (histogram, label `status` ∈ `{success, failure}`) — wall time of one `RunOnce` call.

## Running these jobs

`RunOnce(ctx)` is the one shape both packages expose to any caller — one cycle, no looping, no leadership awareness. Neither job schedules itself: `cmd/api` owns the single loop that turns `RunOnce` into a periodic job (`runPeriodically` — one cycle immediately, then one per `sync_interval`/`interval`, jittered up to 20% so replicas started together don't run their cycles in lockstep), and wires that same loop up two different ways depending on `-database-provider`:

- **PostgreSQL**: the loop runs under `Elector.Run`, so only the current leader runs cycles — see [docs/leader-election.md](leader-election.md) for leadership hand-off. The interval comes from `config.DefaultConfig.<Inventory|Retention>.SyncInterval`/`Interval`, the same value each job's own `NewSyncer`/`NewWorker` already validated as positive at construction.
- **Everything else (SQLite)**: the identical loop, run directly — there's no leader to coordinate since SQLite mode is single-instance-per-file by deployment convention.

Leadership therefore governs *who* runs cycles, never how often they run: one scheduler, one jitter rule, one place to change either. Neither `internal/inventory` nor `internal/retention` imports `internal/leaderelection`, and neither owns a ticker; `cmd/api` is the only place leadership and scheduling meet, through the small `periodicJob` interface (`RunOnce` alone) declared in `cmd/api` itself, the consumer — see `cmd/api/periodic_job.go`.

## Constraints and invariants

- `RunOnce` never loops and never learns who's calling it — scheduling and leadership are entirely the caller's concern. A job that grows a ticker of its own puts a second scheduling rule in the codebase; there is exactly one, and it lives in `cmd/api`.
- Only one of the inventory syncer's step orderings is a data constraint: step 2's refresh reads `metrics_catalog` as the `FROM` table of its own `INSERT`, and `metrics_usage_summary.name` is a foreign key into it, so the summary can only ever see what the catalog step already committed. Step 3 writes `metrics_job_index`, which references neither, and reads nothing either of them writes - it runs last, and only when steps 1-2 succeed, as a deliberate choice not to spend a fan-out across every Prometheus job when the cheap steps already say something is wrong upstream. Anyone reordering these, or running them concurrently, needs that distinction: steps 1-2 are one unit, step 3 is separable.
- Every step within one `RunOnce` call ultimately derives its context from the same `ctx` `RunOnce` was given — canceling that one context reaches every step still in flight, no matter how deep.
- `run_timeout` must stay a true upper bound on `RunOnce`'s worst-case wall time, or the lease strategy's budget enforcement either kills healthy runs early or stops meaning anything. For the inventory syncer that's not just documentation — `NewSyncer` enforces it at construction, widening `run_timeout` to the sum of every enabled step's own timeout plus their overrun allowance when the configured value falls short (see [Time budget](#time-budget)); adding a step, or a new independent sub-timeout, without adding it to that floor would reopen the exact gap https://github.com/nicolastakashi/prom-analytics-proxy/issues/572 fixed.
- Work a step runs concurrently with itself (only the inventory syncer's per-job fan-out within step 3) is still bounded per unit — concurrency changes throughput, not the timeout each unit of work gets.
