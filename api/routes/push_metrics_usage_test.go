package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/nicolastakashi/prom-analytics-proxy/internal/db"
	metricsUsageV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// usageWritesProvider records each rules and dashboard usage write.
type usageWritesProvider struct {
	db.Provider
	rules      [][]db.RulesUsage
	dashboards [][]db.DashboardUsage
}

func (p *usageWritesProvider) InsertRulesUsage(_ context.Context, rules []db.RulesUsage) error {
	p.rules = append(p.rules, rules)
	return nil
}

func (p *usageWritesProvider) InsertDashboardUsage(_ context.Context, dashboards []db.DashboardUsage) error {
	p.dashboards = append(p.dashboards, dashboards)
	return nil
}

// TestPushMetricsUsage_WritesThePayloadOnce: a collector pushes every metric
// of its Prometheus in one request; that's one write per table, not one per
// metric.
func TestPushMetricsUsage_WritesThePayloadOnce(t *testing.T) {
	t.Parallel()
	rule := metricsUsageV1.RuleUsage{PromLink: "https://source-a.example.com", GroupName: "g", Name: "HighLoad", Expression: "up == 0"}
	payload := map[string]*metricsUsageV1.MetricUsage{
		"up":                  {AlertRules: metricsUsageV1.NewSet(rule)},
		"http_requests_total": {RecordingRules: metricsUsageV1.NewSet(metricsUsageV1.RuleUsage{GroupName: "g", Name: "job:rps", Expression: "sum(rate(http_requests_total[5m]))"})},
		"jvm_threads":         {Dashboards: metricsUsageV1.NewSet(metricsUsageV1.DashboardUsage{ID: "uid-a", Name: "JVM", URL: "https://grafana.example.com/d/uid-a"})},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	p := &usageWritesProvider{}
	w := httptest.NewRecorder()
	newTestRouter(t, p).ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/metrics", bytes.NewReader(body)))

	require.Equal(t, 200, w.Code, w.Body.String())
	require.Len(t, p.rules, 1, "one rules write for the whole payload")
	assert.ElementsMatch(t, []db.RulesUsage{
		{Serie: "up", GroupName: "g", Name: "HighLoad", Expression: "up == 0", PromLink: "https://source-a.example.com", Kind: string(db.RuleUsageKindAlert)},
		{Serie: "http_requests_total", GroupName: "g", Name: "job:rps", Expression: "sum(rate(http_requests_total[5m]))", Kind: string(db.RuleUsageKindRecord)},
	}, p.rules[0])
	require.Len(t, p.dashboards, 1, "one dashboard write for the whole payload")
	assert.Equal(t, []db.DashboardUsage{
		{Serie: "jvm_threads", Id: "uid-a", Name: "JVM", URL: "https://grafana.example.com/d/uid-a"},
	}, p.dashboards[0])
}
