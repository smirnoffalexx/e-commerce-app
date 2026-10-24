package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLocker(t *testing.T) (*CheckoutLocker, *miniredis.Miniredis) {
	t.Helper()
	srv := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewCheckoutLocker(client), srv
}

func TestCheckoutLocker_IsExclusiveUntilReleased(t *testing.T) {
	l, _ := newLocker(t)
	ctx := context.Background()

	token, ok, err := l.Acquire(ctx, "user:key", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	_, ok, err = l.Acquire(ctx, "user:key", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok)

	_, ok, err = l.Acquire(ctx, "user:other", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok, "different keys do not block each other")

	require.NoError(t, l.Release(ctx, "user:key", token))
	_, ok, err = l.Acquire(ctx, "user:key", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCheckoutLocker_ExpiresAndIgnoresForeignRelease(t *testing.T) {
	l, srv := newLocker(t)
	ctx := context.Background()

	stale, ok, err := l.Acquire(ctx, "user:key", time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	srv.FastForward(2 * time.Second)
	_, ok, err = l.Acquire(ctx, "user:key", time.Minute)
	require.NoError(t, err)
	require.True(t, ok, "lock of a crashed request expires")

	require.NoError(t, l.Release(ctx, "user:key", stale))
	_, ok, err = l.Acquire(ctx, "user:key", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok, "expired holder must not release the new holder's lock")
}

func TestCheckoutLocker_ReportsRedisErrors(t *testing.T) {
	l, srv := newLocker(t)
	srv.Close()

	_, _, err := l.Acquire(context.Background(), "user:key", time.Minute)
	assert.Error(t, err)
}
