package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRulesUsagePromLink checks a rule's link to its Prometheus is stored,
// follows the latest push - it isn't part of the rule's identity - and comes
// back with when the rule was last seen.
func testRulesUsagePromLink(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()
	start := time.Now().UTC().Add(-time.Minute)

	rule := RulesUsage{Serie: "up", GroupName: "group-a", Name: "TargetDown", Expression: "up == 0", Kind: string(RuleUsageKindAlert)}
	rule.PromLink = "https://prometheus.example.com/graph?g0.expr=up"
	mustInsertRules(t, p, []RulesUsage{rule})
	rule.PromLink = "https://prometheus-b.example.com/graph?g0.expr=up"
	mustInsertRules(t, p, []RulesUsage{rule})

	res, err := p.GetRulesUsage(ctx, RulesUsageParams{
		Serie: "up", Kind: string(RuleUsageKindAlert), Page: 1, PageSize: 10,
		TimeRange: presenceWindow(t, start),
	})
	require.NoError(t, err)
	rows := rowsOf[RulesUsage](t, res)
	require.Len(t, rows, 1, "a new link doesn't make it a new rule")
	assert.Equal(t, "https://prometheus-b.example.com/graph?g0.expr=up", rows[0].PromLink)
	assert.False(t, rows[0].LastSeenAt.Before(rows[0].CreatedAt), "last seen is never before created")
}

func TestPostgreSQL_RulesUsage_PromLink(t *testing.T) {
	t.Parallel()
	testRulesUsagePromLink(t, postgreSQLTestBackend)
}

func TestSQLite_RulesUsage_PromLink(t *testing.T) {
	t.Parallel()
	testRulesUsagePromLink(t, sqliteTestBackend)
}
