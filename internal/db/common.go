package db

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"
)

// ExecuteQuery is a helper function to execute a query with error handling
func ExecuteQuery(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, QueryError(err, "executing query", query)
	}
	return rows, nil
}

// CloseResource safely closes a resource and logs any errors
func CloseResource(closer io.Closer) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil {
		// Log the error but don't panic
		slog.Error("Error closing resource", "error", err)
	}
}

// ScanRows scans every row of rows with scan, then closes rows. It returns an
// empty slice rather than nil when there are no rows.
func ScanRows[T any](rows *sql.Rows, scan func(*sql.Rows) (T, error)) ([]T, error) {
	defer CloseResource(rows)

	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, ErrorWithOperation(err, "scanning row")
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrorWithOperation(err, "row iteration")
	}
	return out, nil
}

// ScanSingleRow scans the first row of rows into dest, then closes rows. It
// returns ErrNoResults when there is no row.
func ScanSingleRow(rows *sql.Rows, dest ...interface{}) error {
	defer CloseResource(rows)

	if !rows.Next() {
		return ErrNoResults
	}

	if err := rows.Scan(dest...); err != nil {
		return ErrorWithOperation(err, "scanning row")
	}

	if err := rows.Err(); err != nil {
		return ErrorWithOperation(err, "row iteration")
	}

	return nil
}

// warnIfSummaryRefreshWasNoOp logs when RefreshMetricsUsageSummary's INSERT
// touched zero rows while metrics_catalog is non-empty. That combination
// means every catalog row failed the last_synced_at freshness filter (see
// https://github.com/nicolastakashi/prom-analytics-proxy/issues/579) - most
// likely metadata sync has stopped advancing last_synced_at (disabled, or a
// seen_ttl raised above inventory.time_window) - which otherwise surfaces as
// a silently frozen summary with no error, the same failure class that issue
// was about.
func warnIfSummaryRefreshWasNoOp(ctx context.Context, db *sql.DB, rowsAffected int64, dialect string) {
	if rowsAffected > 0 {
		return
	}
	var catalogNonEmpty bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM metrics_catalog)`).Scan(&catalogNonEmpty); err != nil {
		return // best-effort diagnostic; don't fail the refresh over it
	}
	if catalogNonEmpty {
		slog.Warn("inventory: refresh summary touched no rows despite a non-empty catalog; "+
			"every row failed the last_synced_at freshness filter - check that metadata sync is advancing last_synced_at",
			"dialect", dialect)
	}
}

// GetInterval returns the appropriate interval string for time-based queries
func GetInterval(from, to time.Time, dialect string) string {
	duration := to.Sub(from)
	var interval string

	switch {
	case duration <= 2*time.Hour:
		interval = "1 minute"
	case duration <= 6*time.Hour:
		interval = "5 minutes"
	case duration <= 24*time.Hour:
		interval = "15 minutes"
	case duration <= 7*24*time.Hour:
		interval = "1 hour"
	case duration <= 30*24*time.Hour:
		interval = "6 hours"
	case duration <= 90*24*time.Hour:
		interval = "1 day"
	default:
		interval = "1 day"
	}

	if dialect == "postgresql" {
		return interval
	}

	// SQLite has a different interval syntax
	switch interval {
	case "1 minute":
		return "+1 minutes"
	case "5 minutes":
		return "+5 minutes"
	case "15 minutes":
		return "+15 minutes"
	case "1 hour":
		return "+1 hours"
	case "6 hours":
		return "+6 hours"
	case "1 day":
		return "+1 days"
	default:
		return "+1 days"
	}
}

// sortedUniqueNames is what a single-statement upsert keyed by name needs:
// no name twice, and every writer locking rows in the same order.
func sortedUniqueNames(names []string) []string {
	out := slices.Clone(names)
	slices.Sort(out)
	return slices.Compact(out)
}

// groupCatalogSources folds (name, source) rows, ordered by name then
// source, into each name's sorted list of sources.
func groupCatalogSources(rows *sql.Rows) (map[string][]string, error) {
	type pair struct{ name, source string }
	pairs, err := ScanRows(rows, func(r *sql.Rows) (pair, error) {
		var p pair
		err := r.Scan(&p.name, &p.source)
		return p, err
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string)
	for _, p := range pairs {
		out[p.name] = append(out[p.name], p.source)
	}
	return out, nil
}

// sortedUniqueServiceItems de-duplicates and orders service index items by
// their key, so a single-statement upsert never touches a row twice and
// concurrent writers lock rows in the same order.
func sortedUniqueServiceItems(items []MetricServiceIndexItem) []MetricServiceIndexItem {
	out := slices.Clone(items)
	slices.SortFunc(out, func(a, b MetricServiceIndexItem) int {
		if c := strings.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		if c := strings.Compare(a.Service, b.Service); c != 0 {
			return c
		}
		return strings.Compare(a.Job, b.Job)
	})
	return slices.Compact(out)
}

// cardinalityArrays flattens a series-per-metric map into parallel slices,
// sorted by name so concurrent writers lock rows in the same order.
func cardinalityArrays(series map[string]int) ([]string, []int64) {
	names := make([]string, 0, len(series))
	for name := range series {
		names = append(names, name)
	}
	slices.Sort(names)
	counts := make([]int64, len(names))
	for i, name := range names {
		counts[i] = int64(series[name])
	}
	return names, counts
}

// groupCardinality folds (name, source, series) rows into series per source
// per name.
func groupCardinality(rows *sql.Rows) (map[string]map[string]int, error) {
	type row struct {
		name, source string
		series       int
	}
	all, err := ScanRows(rows, func(r *sql.Rows) (row, error) {
		var x row
		err := r.Scan(&x.name, &x.source, &x.series)
		return x, err
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string]map[string]int)
	for _, x := range all {
		if out[x.name] == nil {
			out[x.name] = make(map[string]int)
		}
		out[x.name][x.source] = x.series
	}
	return out, nil
}

// serviceIndexChunk bounds how many service index rows one statement writes.
const serviceIndexChunk = 5000

// cardinalityChunk bounds how many cardinality rows one statement writes.
const cardinalityChunk = 5000

// UpstreamSeriesSource is the source GetMetricsCardinality reports a
// metric's job index series under - counted in the upstream - when no
// metadata source counted it.
const UpstreamSeriesSource = "upstream"

// nullSeries stores an unknown (0) series count as NULL.
func nullSeries(n int64) any {
	if n <= 0 {
		return nil
	}
	return n
}
