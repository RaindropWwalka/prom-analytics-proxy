package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testJobLastRun(t *testing.T, b testBackend) {
	p := b.newProvider(t)
	ctx := context.Background()

	_, ok, err := p.GetJobLastRun(ctx, "job-a")
	require.NoError(t, err)
	assert.False(t, ok, "never ran")

	first := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	require.NoError(t, p.SetJobLastRun(ctx, "job-a", first))
	require.NoError(t, p.SetJobLastRun(ctx, "job-a", first.Add(time.Hour)))
	require.NoError(t, p.SetJobLastRun(ctx, "job-b", first))

	got, ok, err := p.GetJobLastRun(ctx, "job-a")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, first.Add(time.Hour), got, "the latest run wins")
}

func TestPostgreSQL_JobLastRun(t *testing.T) {
	t.Parallel()
	testJobLastRun(t, postgreSQLTestBackend)
}

func TestSQLite_JobLastRun(t *testing.T) {
	t.Parallel()
	testJobLastRun(t, sqliteTestBackend)
}
