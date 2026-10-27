package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testQueriesBySerieNameOrigins checks a metric's queries say who ran them,
// from the headers stored with each run, and when they last ran.
func testQueriesBySerieNameOrigins(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	now := time.Now().UTC().Truncate(time.Second)

	run := func(minute int, headers map[string]string) Query {
		q := instantQuery(now.Add(-time.Duration(minute)*time.Minute), "up")
		q.HTTPHeaders = headers
		return q
	}
	mustInsertQueries(t, p, []Query{
		run(1, map[string]string{"x-dashboard-uid": "dash-a", "x-dashboard-title": "Service overview"}),
		run(2, map[string]string{"x-dashboard-uid": "dash-b"}),
		run(3, map[string]string{"fromalert": "true", "x-rule-name": "TargetDown", "x-dashboard-uid": "dash-a"}),
		run(4, map[string]string{"user-agent": "curl/8.7.1"}),
		run(5, map[string]string{"user-agent": "curl/8.7.1", "x-client-name": "team-a/report"}),
		run(6, nil),
	})

	res, err := p.GetQueriesBySerieName(context.Background(), QueriesBySerieNameParams{
		SerieName: "up", TimeRange: mustTimeRange(t, now.Add(-time.Hour), now.Add(time.Minute)),
		Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	rows := rowsOf[QueriesBySerieNameResult](t, res)
	require.Len(t, rows, 1, "one query expression")
	r := rows[0]

	assert.Equal(t, 6, r.Executions)
	assert.Equal(t, 2, r.DashboardRuns, "an alert built from a dashboard counts as the alert")
	assert.Equal(t, 1, r.AlertRuns)
	assert.Equal(t, []string{"Service overview", "dash-b"}, r.Dashboards, "by title, else by uid")
	assert.Equal(t, []string{"TargetDown"}, r.Rules)
	assert.Equal(t, []string{"curl/8.7.1", "team-a/report"}, r.Clients, "client name over user agent")
	assert.WithinDuration(t, now.Add(-time.Minute), r.TS, time.Second, "when it last ran")
}

func TestPostgreSQL_GetQueriesBySerieName_Origins(t *testing.T) {
	t.Parallel()
	testQueriesBySerieNameOrigins(t, postgreSQLTestBackend)
}

func TestSQLite_GetQueriesBySerieName_Origins(t *testing.T) {
	t.Parallel()
	testQueriesBySerieNameOrigins(t, sqliteTestBackend)
}
