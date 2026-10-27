package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testQueryExecutionsFilters seeds one execution per kind of row the
// filters tell apart - each marked by its sample count - and checks which
// ones every filter keeps.
func testQueryExecutionsFilters(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	now := time.Now().UTC().Truncate(time.Second)

	execution := func(minute, samples, status int, headers map[string]string) Query {
		q := instantQuery(now.Add(-time.Duration(minute)*time.Minute), "up")
		q.Fingerprint = "fp-filters"
		q.TotalQueryableSamples = samples
		q.StatusCode = status
		q.HTTPHeaders = headers
		return q
	}
	mustInsertQueries(t, p, []Query{
		execution(1, 1, 200, map[string]string{"x-dashboard-uid": "dash-a", "x-panel-id": "3", "user-agent": "Grafana/12.0.0"}),
		execution(2, 2, 200, map[string]string{"fromalert": "true", "x-rule-name": "HighErrorRate", "x-dashboard-uid": "dash-a"}),
		execution(3, 3, 422, map[string]string{"user-agent": "curl/8.7.1"}),
		execution(4, 4, 503, nil),
	})

	samplesFor := func(params QueryExecutionsParams) []int {
		t.Helper()
		params.Fingerprint = "fp-filters"
		params.TimeRange = mustTimeRange(t, now.Add(-time.Hour), now.Add(time.Minute))
		params.Page, params.PageSize = 1, 10
		params.SortBy, params.SortOrder = "samples", "asc"
		res, err := p.GetQueryExecutions(context.Background(), params)
		require.NoError(t, err)
		var samples []int
		for _, r := range rowsOf[QueryExecutionRow](t, res) {
			samples = append(samples, r.Samples)
		}
		return samples
	}

	for _, tc := range []struct {
		name   string
		params QueryExecutionsParams
		want   []int
	}{
		{"no filter", QueryExecutionsParams{}, []int{1, 2, 3, 4}},
		{"status 2xx", QueryExecutionsParams{Status: ExecutionStatus2xx}, []int{1, 2}},
		{"status 4xx", QueryExecutionsParams{Status: ExecutionStatus4xx}, []int{3}},
		{"status 5xx", QueryExecutionsParams{Status: ExecutionStatus5xx}, []int{4}},
		{"status error", QueryExecutionsParams{Status: ExecutionStatusError}, []int{3, 4}},
		{"origin alert", QueryExecutionsParams{Origin: ExecutionOriginAlert}, []int{2}},
		// An alert built from a dashboard panel carries the dashboard's uid too,
		// but it's the alert that ran it.
		{"origin dashboard", QueryExecutionsParams{Origin: ExecutionOriginDashboard}, []int{1}},
		{"origin other", QueryExecutionsParams{Origin: ExecutionOriginOther}, []int{3, 4}},
		{"header value", QueryExecutionsParams{HeaderSearch: "CURL"}, []int{3}},
		{"header name", QueryExecutionsParams{HeaderSearch: "x-rule-name"}, []int{2}},
		{"combined", QueryExecutionsParams{Origin: ExecutionOriginDashboard, Status: ExecutionStatusError}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, samplesFor(tc.params))
		})
	}
}

func TestPostgreSQL_GetQueryExecutions_Filters(t *testing.T) {
	t.Parallel()
	testQueryExecutionsFilters(t, postgreSQLTestBackend)
}

func TestSQLite_GetQueryExecutions_Filters(t *testing.T) {
	t.Parallel()
	testQueryExecutionsFilters(t, sqliteTestBackend)
}
