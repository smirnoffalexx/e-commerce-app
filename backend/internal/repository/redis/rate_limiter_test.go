package redis

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newLimiter(t *testing.T) (*RateLimiter, *clock) {
	t.Helper()
	srv := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	c := &clock{now: time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)}
	l := NewRateLimiter(client)
	l.now = c.Now
	return l, c
}

func TestRateLimiter_BlocksAfterLimit(t *testing.T) {
	l, _ := newLimiter(t)
	ctx := context.Background()

	for i := range 3 {
		allowed, remaining, reset, err := l.Allow(ctx, "k", 3, time.Minute)
		require.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 2-i, remaining)
		assert.Equal(t, time.Minute, reset)
	}

	allowed, remaining, _, err := l.Allow(ctx, "k", 3, time.Minute)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Zero(t, remaining)
}

func TestRateLimiter_SlidingWindow(t *testing.T) {
	l, c := newLimiter(t)
	ctx := context.Background()

	allow := func() bool {
		ok, _, _, err := l.Allow(ctx, "k", 2, time.Minute)
		require.NoError(t, err)
		return ok
	}

	require.True(t, allow())
	c.Advance(30 * time.Second)
	require.True(t, allow())
	require.False(t, allow())

	_, _, reset, err := l.Allow(ctx, "k", 2, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, reset, "slot frees when the oldest request leaves the window")

	c.Advance(30 * time.Second)
	assert.True(t, allow(), "first request left the window")
	assert.False(t, allow(), "second request is still inside the window")

	c.Advance(30 * time.Second)
	assert.True(t, allow())
}

func TestRateLimiter_RejectedRequestsDoNotExtendBlock(t *testing.T) {
	l, c := newLimiter(t)
	ctx := context.Background()

	ok, _, _, err := l.Allow(ctx, "k", 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	for range 10 {
		c.Advance(5 * time.Second)
		ok, _, _, err = l.Allow(ctx, "k", 1, time.Minute)
		require.NoError(t, err)
		require.False(t, ok)
	}

	c.Advance(10 * time.Second)
	ok, _, _, err = l.Allow(ctx, "k", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, ok, "hammering while blocked must not keep the client blocked forever")
}

func TestRateLimiter_KeysAreIsolated(t *testing.T) {
	l, _ := newLimiter(t)
	ctx := context.Background()

	ok, _, _, err := l.Allow(ctx, "auth:ip:1.1.1.1", 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	ok, _, _, err = l.Allow(ctx, "auth:ip:2.2.2.2", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRateLimiter_ConcurrentRequestsNeverExceedLimit(t *testing.T) {
	l, _ := newLimiter(t)
	ctx := context.Background()

	const limit, clients = 10, 50
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for range clients {
		wg.Go(func() {
			ok, _, _, err := l.Allow(ctx, "k", limit, time.Minute)
			if err != nil {
				t.Error(err)
				return
			}
			if ok {
				allowed.Add(1)
			}
		})
	}
	wg.Wait()

	assert.EqualValues(t, limit, allowed.Load())
}

func TestRateLimiter_RedisDown(t *testing.T) {
	srv := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: srv.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	srv.Close()

	_, _, _, err := NewRateLimiter(client).Allow(context.Background(), "k", 1, time.Minute)
	assert.Error(t, err)
}
