package db

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeferredProvider_FailsEveryCallUntilSet walks every Provider method:
// before Set, each returns ErrDatabaseUnavailable rather than panicking.
func TestDeferredProvider_FailsEveryCallUntilSet(t *testing.T) {
	d := reflect.ValueOf(NewDeferredProvider())
	iface := reflect.TypeOf((*Provider)(nil)).Elem()
	for i := range iface.NumMethod() {
		m := iface.Method(i)
		if m.Name == "WithDB" || m.Name == "Close" {
			continue
		}
		method := d.MethodByName(m.Name)
		args := make([]reflect.Value, method.Type().NumIn())
		for j := range args {
			args[j] = reflect.Zero(method.Type().In(j))
		}
		args[0] = reflect.ValueOf(context.Background())
		out := method.Call(args)
		err, _ := out[len(out)-1].Interface().(error)
		assert.ErrorIs(t, err, ErrDatabaseUnavailable, m.Name)
	}

	called := false
	NewDeferredProvider().WithDB(func(*sql.DB) { called = true })
	assert.False(t, called, "WithDB has no database to give yet")
	assert.NoError(t, NewDeferredProvider().Close())
}

func TestDeferredProvider_PassesThroughOnceSet(t *testing.T) {
	p := newTestSQLiteProvider(t)
	d := NewDeferredProvider()
	d.Set(p)

	_, err := d.ListJobs(context.Background())
	assert.NoError(t, err)
}

func TestConnectWithRetry_RetriesUntilConnected(t *testing.T) {
	attempts := 0
	want := NewDeferredProvider()
	got, err := ConnectWithRetry(context.Background(), func(context.Context) (Provider, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("connection refused")
		}
		return want, nil
	}, time.Millisecond, 2*time.Millisecond)

	require.NoError(t, err)
	assert.Same(t, want, got)
	assert.Equal(t, 3, attempts)
}

func TestConnectWithRetry_StopsWithTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ConnectWithRetry(ctx, func(context.Context) (Provider, error) {
		return nil, errors.New("connection refused")
	}, time.Hour, time.Hour)

	assert.ErrorIs(t, err, context.Canceled)
}
