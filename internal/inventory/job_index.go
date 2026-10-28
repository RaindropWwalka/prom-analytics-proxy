package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nicolastakashi/prom-analytics-proxy/internal/db"
	"github.com/prometheus/common/model"
)

// syncJobIndex populates job_index: which metrics each Prometheus job
// produces, used to answer job-scoped "which of this job's metrics are
// unused" queries. Bounded as a whole by jobIndexTimeout - independent of
// every other RunOnce step (see docs/jobs.md) - since its own duration
// otherwise scales with however many jobs Prometheus reports at run time,
// something no per-call timeout alone can bound.
func (s *Syncer) syncJobIndex(ctx context.Context, tr db.TimeRange) error {
	jobIndexCtx, cancel := context.WithTimeout(ctx, s.jobIndexTimeout)
	defer cancel()

	labelCtx, cancelLabels := context.WithTimeout(jobIndexCtx, s.jobIndexLabelTimeout)
	defer cancelLabels()
	jobs, _, _, err := s.promAPI.LabelValues(labelCtx, "job", []string{}, tr.From(), tr.To())
	if err != nil {
		// No series with a job label is a successful, empty response, so
		// every error here - 4xx included - is a failure.
		return fmt.Errorf("fetch job label values: %w", err)
	}

	if len(jobs) == 0 {
		slog.Debug("no job labels found in time range", "from", tr.From(), "to", tr.To())
		return nil
	}

	slog.Info("syncing job index", "jobs_found", len(jobs), "workers", s.jobIndexWorkers, "time_range", tr)

	jobChan := make(chan string, len(jobs))
	errorChan := make(chan error, len(jobs))

	var wg sync.WaitGroup
	for i := 0; i < s.jobIndexWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobChan {
				err := s.processJob(jobIndexCtx, job, tr, workerID)
				errorChan <- err
			}
		}(i)
	}

	jobsProcessed := 0
	for _, jobLabel := range jobs {
		job := string(jobLabel)
		if job != "" {
			jobChan <- job
			jobsProcessed++
		}
	}
	close(jobChan)

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	var successCount, failureCount int
	for err := range errorChan {
		if err != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	slog.Info("job index sync complete",
		"jobs_processed", jobsProcessed,
		"successful", successCount,
		"failed", failureCount)

	if failureCount > successCount/2 {
		return fmt.Errorf("job index sync failed: %d failures out of %d jobs", failureCount, jobsProcessed)
	}

	return nil
}

// processJob queries one job's metric names with their series counts - and,
// with a service label, which service produces each - and commits them. Both calls are
// bounded by jobIndexPerJobTimeout, so a job stuck on an upstream query or a
// database lock costs its own budget rather than the whole step's, and the
// worker is back on the fan-out channel either way.
func (s *Syncer) processJob(ctx context.Context, job string, tr db.TimeRange, workerID int) error {
	jobCtx, cancelJob := context.WithTimeout(ctx, s.jobIndexPerJobTimeout)
	defer cancelJob()

	by := "__name__"
	if s.serviceLabel != "" {
		by += ", " + s.serviceLabel
	}
	// count, not group: the same scan of the job's series also tells how
	// many each metric has, which the catalog falls back on for metrics
	// outside every source's /api/v1/status/tsdb top-N.
	query := fmt.Sprintf(`count({job="%s", __name__=~".+"}) by (%s)`, job, by)
	result, _, _, err := s.promAPI.Query(jobCtx, query, tr.To())
	if err != nil {
		slog.Warn("failed to query metrics for job", "job", job, "worker", workerID, "query", query, "err", err)
		return err
	}

	metricSeries := make(map[string]int64)
	var services []db.MetricServiceIndexItem

	switch v := result.(type) {
	case model.Vector:
		for _, sample := range v {
			metricName, ok := sample.Metric["__name__"]
			if !ok {
				continue
			}
			metricSeries[string(metricName)] += int64(sample.Value)
			if s.serviceLabel == "" {
				continue
			}
			if service := sample.Metric[model.LabelName(s.serviceLabel)]; service != "" {
				services = append(services, db.MetricServiceIndexItem{Name: string(metricName), Service: string(service), Job: job})
			}
		}
	default:
		slog.Debug("unexpected query result type", "job", job, "worker", workerID, "type", fmt.Sprintf("%T", result))
	}

	var items []db.MetricJobIndexItem
	for name, series := range metricSeries {
		items = append(items, db.MetricJobIndexItem{Name: name, Job: job, Series: series})
	}

	if len(items) > 0 {
		if err := s.dbProvider.UpsertMetricsJobIndex(jobCtx, items); err != nil {
			slog.Error("failed to upsert metrics for job", "job", job, "worker", workerID, "metrics", len(items), "err", err)
			return fmt.Errorf("upsert job %s: %w", job, err)
		}
		slog.Debug("processed job", "job", job, "worker", workerID, "metrics", len(items))
	}
	if len(services) > 0 {
		if err := s.dbProvider.UpsertMetricsServiceIndex(jobCtx, services, time.Now().Add(-s.serviceIndexRefreshInterval)); err != nil {
			slog.Error("failed to upsert services for job", "job", job, "worker", workerID, "services", len(services), "err", err)
			return fmt.Errorf("upsert services of job %s: %w", job, err)
		}
	}

	return nil
}
