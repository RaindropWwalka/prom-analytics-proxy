package db

import (
	"context"
	"database/sql"
	"fmt"
)

// The unused-metrics analysis, one static statement per grouping and
// backend. Rows whose last sync is older than SyncedSince are left out:
// the summary no longer refreshes them, so their usage isn't current.
//
// Job comes from metrics_job_index, service from metrics_service_index and
// source from metrics_catalog_sources, none of which knows the others: a
// metric of job "api" reported by two sources counts under both (api,
// source) pairs, even if only one of them scrapes that job. A metric a
// service produces under several jobs counts once per service.
//
// Series come from metrics_cardinality: per (metric, source) when grouped by
// source, else the metric's series summed over sources. A metric outside
// every source's recorded top-N counts as 0 series, and a metric shared by
// several jobs or services counts its whole series under each of them.
const (
	pgUnusedByJobSQL = `
        SELECT j.job, '' AS service, '' AS source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END),
               COALESCE(SUM(k.series), 0),
               COALESCE(SUM(CASE WHEN s.is_unused THEN k.series ELSE 0 END), 0)
        FROM metrics_job_index j
        JOIN metrics_catalog c ON c.name = j.name
        JOIN metrics_usage_summary s ON s.name = j.name
        LEFT JOIN (SELECT name, SUM(series) AS series FROM metrics_cardinality GROUP BY name) k ON k.name = c.name
        WHERE c.last_synced_at >= $1
          AND ($2 = '' OR j.job = $2)
        GROUP BY j.job
        ORDER BY 5 DESC, 1`
	pgUnusedBySourceSQL = `
        SELECT '' AS job, '' AS service, src.source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END),
               COALESCE(SUM(k.series), 0),
               COALESCE(SUM(CASE WHEN s.is_unused THEN k.series ELSE 0 END), 0)
        FROM metrics_catalog_sources src
        JOIN metrics_catalog c ON c.name = src.name
        JOIN metrics_usage_summary s ON s.name = src.name
        LEFT JOIN metrics_cardinality k ON k.name = src.name AND k.source = src.source
        WHERE c.last_synced_at >= $1
          AND ($2 = '' OR src.source = $2)
        GROUP BY src.source
        ORDER BY 5 DESC, 3`
	pgUnusedByJobSourceSQL = `
        SELECT j.job, '' AS service, src.source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END),
               COALESCE(SUM(k.series), 0),
               COALESCE(SUM(CASE WHEN s.is_unused THEN k.series ELSE 0 END), 0)
        FROM metrics_job_index j
        JOIN metrics_catalog_sources src ON src.name = j.name
        JOIN metrics_catalog c ON c.name = j.name
        JOIN metrics_usage_summary s ON s.name = j.name
        LEFT JOIN metrics_cardinality k ON k.name = src.name AND k.source = src.source
        WHERE c.last_synced_at >= $1
          AND ($2 = '' OR j.job = $2)
          AND ($3 = '' OR src.source = $3)
        GROUP BY j.job, src.source
        ORDER BY 5 DESC, 1, 3`
	pgUnusedByServiceSQL = `
        SELECT '' AS job, svc.service, '' AS source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END),
               COALESCE(SUM(k.series), 0),
               COALESCE(SUM(CASE WHEN s.is_unused THEN k.series ELSE 0 END), 0)
        FROM (SELECT DISTINCT name, service FROM metrics_service_index) svc
        JOIN metrics_catalog c ON c.name = svc.name
        JOIN metrics_usage_summary s ON s.name = svc.name
        LEFT JOIN (SELECT name, SUM(series) AS series FROM metrics_cardinality GROUP BY name) k ON k.name = c.name
        WHERE c.last_synced_at >= $1
          AND ($2 = '' OR svc.service = $2)
        GROUP BY svc.service
        ORDER BY 5 DESC, 2`
	pgUnusedByServiceSourceSQL = `
        SELECT '' AS job, svc.service, src.source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END),
               COALESCE(SUM(k.series), 0),
               COALESCE(SUM(CASE WHEN s.is_unused THEN k.series ELSE 0 END), 0)
        FROM (SELECT DISTINCT name, service FROM metrics_service_index) svc
        JOIN metrics_catalog_sources src ON src.name = svc.name
        JOIN metrics_catalog c ON c.name = svc.name
        JOIN metrics_usage_summary s ON s.name = svc.name
        LEFT JOIN metrics_cardinality k ON k.name = src.name AND k.source = src.source
        WHERE c.last_synced_at >= $1
          AND ($2 = '' OR svc.service = $2)
          AND ($3 = '' OR src.source = $3)
        GROUP BY svc.service, src.source
        ORDER BY 5 DESC, 2, 3`
	sqliteUnusedByJobSQL = `
        SELECT j.job, '' AS service, '' AS source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END),
               COALESCE(SUM(k.series), 0),
               COALESCE(SUM(CASE WHEN s.is_unused THEN k.series ELSE 0 END), 0)
        FROM metrics_job_index j
        JOIN metrics_catalog c ON c.name = j.name
        JOIN metrics_usage_summary s ON s.name = j.name
        LEFT JOIN (SELECT name, SUM(series) AS series FROM metrics_cardinality GROUP BY name) k ON k.name = c.name
        WHERE c.last_synced_at >= ?
          AND (? = '' OR j.job = ?)
        GROUP BY j.job
        ORDER BY 5 DESC, 1`
	sqliteUnusedBySourceSQL = `
        SELECT '' AS job, '' AS service, src.source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END),
               COALESCE(SUM(k.series), 0),
               COALESCE(SUM(CASE WHEN s.is_unused THEN k.series ELSE 0 END), 0)
        FROM metrics_catalog_sources src
        JOIN metrics_catalog c ON c.name = src.name
        JOIN metrics_usage_summary s ON s.name = src.name
        LEFT JOIN metrics_cardinality k ON k.name = src.name AND k.source = src.source
        WHERE c.last_synced_at >= ?
          AND (? = '' OR src.source = ?)
        GROUP BY src.source
        ORDER BY 5 DESC, 3`
	sqliteUnusedByJobSourceSQL = `
        SELECT j.job, '' AS service, src.source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END),
               COALESCE(SUM(k.series), 0),
               COALESCE(SUM(CASE WHEN s.is_unused THEN k.series ELSE 0 END), 0)
        FROM metrics_job_index j
        JOIN metrics_catalog_sources src ON src.name = j.name
        JOIN metrics_catalog c ON c.name = j.name
        JOIN metrics_usage_summary s ON s.name = j.name
        LEFT JOIN metrics_cardinality k ON k.name = src.name AND k.source = src.source
        WHERE c.last_synced_at >= ?
          AND (? = '' OR j.job = ?)
          AND (? = '' OR src.source = ?)
        GROUP BY j.job, src.source
        ORDER BY 5 DESC, 1, 3`
	sqliteUnusedByServiceSQL = `
        SELECT '' AS job, svc.service, '' AS source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END),
               COALESCE(SUM(k.series), 0),
               COALESCE(SUM(CASE WHEN s.is_unused THEN k.series ELSE 0 END), 0)
        FROM (SELECT DISTINCT name, service FROM metrics_service_index) svc
        JOIN metrics_catalog c ON c.name = svc.name
        JOIN metrics_usage_summary s ON s.name = svc.name
        LEFT JOIN (SELECT name, SUM(series) AS series FROM metrics_cardinality GROUP BY name) k ON k.name = c.name
        WHERE c.last_synced_at >= ?
          AND (? = '' OR svc.service = ?)
        GROUP BY svc.service
        ORDER BY 5 DESC, 2`
	sqliteUnusedByServiceSourceSQL = `
        SELECT '' AS job, svc.service, src.source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END),
               COALESCE(SUM(k.series), 0),
               COALESCE(SUM(CASE WHEN s.is_unused THEN k.series ELSE 0 END), 0)
        FROM (SELECT DISTINCT name, service FROM metrics_service_index) svc
        JOIN metrics_catalog_sources src ON src.name = svc.name
        JOIN metrics_catalog c ON c.name = svc.name
        JOIN metrics_usage_summary s ON s.name = svc.name
        LEFT JOIN metrics_cardinality k ON k.name = src.name AND k.source = src.source
        WHERE c.last_synced_at >= ?
          AND (? = '' OR svc.service = ?)
          AND (? = '' OR src.source = ?)
        GROUP BY svc.service, src.source
        ORDER BY 5 DESC, 2, 3`
)

func (p *PostGreSQLProvider) AnalyzeUnusedMetrics(ctx context.Context, params UnusedAnalysisParams) ([]UnusedAnalysisRow, error) {
	since := params.SyncedSince.UTC().Format(ISOTimeFormat)
	switch params.GroupBy {
	case UnusedAnalysisByJob:
		return queryUnusedAnalysis(ctx, p.db, pgUnusedByJobSQL, since, params.Job)
	case UnusedAnalysisBySource:
		return queryUnusedAnalysis(ctx, p.db, pgUnusedBySourceSQL, since, params.Source)
	case UnusedAnalysisByJobSource:
		return queryUnusedAnalysis(ctx, p.db, pgUnusedByJobSourceSQL, since, params.Job, params.Source)
	case UnusedAnalysisByService:
		return queryUnusedAnalysis(ctx, p.db, pgUnusedByServiceSQL, since, params.Service)
	case UnusedAnalysisByServiceSource:
		return queryUnusedAnalysis(ctx, p.db, pgUnusedByServiceSourceSQL, since, params.Service, params.Source)
	}
	return nil, fmt.Errorf("unknown grouping %q", params.GroupBy)
}

func (p *SQLiteProvider) AnalyzeUnusedMetrics(ctx context.Context, params UnusedAnalysisParams) ([]UnusedAnalysisRow, error) {
	since := params.SyncedSince.UTC().Format(SQLiteTimeFormat)
	switch params.GroupBy {
	case UnusedAnalysisByJob:
		return queryUnusedAnalysis(ctx, p.db, sqliteUnusedByJobSQL, since, params.Job, params.Job)
	case UnusedAnalysisBySource:
		return queryUnusedAnalysis(ctx, p.db, sqliteUnusedBySourceSQL, since, params.Source, params.Source)
	case UnusedAnalysisByJobSource:
		return queryUnusedAnalysis(ctx, p.db, sqliteUnusedByJobSourceSQL, since, params.Job, params.Job, params.Source, params.Source)
	case UnusedAnalysisByService:
		return queryUnusedAnalysis(ctx, p.db, sqliteUnusedByServiceSQL, since, params.Service, params.Service)
	case UnusedAnalysisByServiceSource:
		return queryUnusedAnalysis(ctx, p.db, sqliteUnusedByServiceSourceSQL, since, params.Service, params.Service, params.Source, params.Source)
	}
	return nil, fmt.Errorf("unknown grouping %q", params.GroupBy)
}

func queryUnusedAnalysis(ctx context.Context, db *sql.DB, query string, args ...any) ([]UnusedAnalysisRow, error) {
	rows, err := ExecuteQuery(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	return ScanRows(rows, func(r *sql.Rows) (UnusedAnalysisRow, error) {
		var row UnusedAnalysisRow
		err := r.Scan(&row.Job, &row.Service, &row.Source, &row.Total, &row.Unused, &row.Series, &row.UnusedSeries)
		return row, err
	})
}
