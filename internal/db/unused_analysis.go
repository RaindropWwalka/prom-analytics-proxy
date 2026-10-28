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
// Job comes from metrics_job_index and source from metrics_catalog_sources,
// neither of which knows the other: a metric of job "api" reported by two
// sources counts under both (api, source) pairs, even if only one of them
// scrapes that job.
const (
	pgUnusedByJobSQL = `
        SELECT j.job, '' AS source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END)
        FROM metrics_job_index j
        JOIN metrics_catalog c ON c.name = j.name
        JOIN metrics_usage_summary s ON s.name = j.name
        WHERE c.last_synced_at >= $1
          AND ($2 = '' OR j.job = $2)
        GROUP BY j.job
        ORDER BY 4 DESC, 1`
	pgUnusedBySourceSQL = `
        SELECT '' AS job, src.source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END)
        FROM metrics_catalog_sources src
        JOIN metrics_catalog c ON c.name = src.name
        JOIN metrics_usage_summary s ON s.name = src.name
        WHERE c.last_synced_at >= $1
          AND ($2 = '' OR src.source = $2)
        GROUP BY src.source
        ORDER BY 4 DESC, 2`
	pgUnusedByJobSourceSQL = `
        SELECT j.job, src.source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END)
        FROM metrics_job_index j
        JOIN metrics_catalog_sources src ON src.name = j.name
        JOIN metrics_catalog c ON c.name = j.name
        JOIN metrics_usage_summary s ON s.name = j.name
        WHERE c.last_synced_at >= $1
          AND ($2 = '' OR j.job = $2)
          AND ($3 = '' OR src.source = $3)
        GROUP BY j.job, src.source
        ORDER BY 4 DESC, 1, 2`

	sqliteUnusedByJobSQL = `
        SELECT j.job, '' AS source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END)
        FROM metrics_job_index j
        JOIN metrics_catalog c ON c.name = j.name
        JOIN metrics_usage_summary s ON s.name = j.name
        WHERE c.last_synced_at >= ?
          AND (? = '' OR j.job = ?)
        GROUP BY j.job
        ORDER BY 4 DESC, 1`
	sqliteUnusedBySourceSQL = `
        SELECT '' AS job, src.source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END)
        FROM metrics_catalog_sources src
        JOIN metrics_catalog c ON c.name = src.name
        JOIN metrics_usage_summary s ON s.name = src.name
        WHERE c.last_synced_at >= ?
          AND (? = '' OR src.source = ?)
        GROUP BY src.source
        ORDER BY 4 DESC, 2`
	sqliteUnusedByJobSourceSQL = `
        SELECT j.job, src.source, COUNT(*),
               SUM(CASE WHEN s.is_unused THEN 1 ELSE 0 END)
        FROM metrics_job_index j
        JOIN metrics_catalog_sources src ON src.name = j.name
        JOIN metrics_catalog c ON c.name = j.name
        JOIN metrics_usage_summary s ON s.name = j.name
        WHERE c.last_synced_at >= ?
          AND (? = '' OR j.job = ?)
          AND (? = '' OR src.source = ?)
        GROUP BY j.job, src.source
        ORDER BY 4 DESC, 1, 2`
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
		err := r.Scan(&row.Job, &row.Source, &row.Total, &row.Unused)
		return row, err
	})
}
