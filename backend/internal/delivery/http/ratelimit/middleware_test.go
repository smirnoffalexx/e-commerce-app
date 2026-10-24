package ratelimit_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/delivery/http/middleware"
	"e-commerce-backend/internal/delivery/http/ratelimit"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeLimiter struct {
	allowed   bool
	remaining int
	resetIn   time.Duration
	err       error
	keys      []string
}

func (f *fakeLimiter) Allow(_ context.Context, key string, _ int, _ time.Duration) (bool, int, time.Duration, error) {
	f.keys = append(f.keys, key)
	return f.allowed, f.remaining, f.resetIn, f.err
}

func serve(t *testing.T, l ratelimit.Limiter, p ratelimit.Policy, trusted []string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	require.NoError(t, r.SetTrustedProxies(trusted))
	r.Use(middleware.RequestID(), ratelimit.Middleware(l, p))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:51234"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

var policy = ratelimit.Policy{Name: "auth", Limit: 10, Window: time.Minute}

func TestMiddleware_AllowedSetsHeaders(t *testing.T) {
	l := &fakeLimiter{allowed: true, remaining: 7, resetIn: 1500 * time.Millisecond}

	w := serve(t, l, policy, nil, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "10", w.Header().Get("RateLimit-Limit"))
	assert.Equal(t, "7", w.Header().Get("RateLimit-Remaining"))
	assert.Equal(t, "2", w.Header().Get("RateLimit-Reset"), "reset is rounded up")
	assert.Equal(t, []string{"auth:ip:10.0.0.5"}, l.keys)
}

func TestMiddleware_Blocked(t *testing.T) {
	l := &fakeLimiter{allowed: false, resetIn: 42 * time.Second}

	w := serve(t, l, policy, nil, nil)

	require.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "42", w.Header().Get("Retry-After"))
	assert.Contains(t, w.Body.String(), `"code":"rate_limited"`)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
}

func TestMiddleware_LimiterFailure(t *testing.T) {
	down := errors.New("redis: connection refused")

	t.Run("fail closed", func(t *testing.T) {
		w := serve(t, &fakeLimiter{err: down}, policy, nil, nil)
		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), `"code":"rate_limiter_unavailable"`)
		assert.NotContains(t, w.Body.String(), "redis")
	})

	t.Run("fail open", func(t *testing.T) {
		open := policy
		open.FailOpen = true
		w := serve(t, &fakeLimiter{err: down}, open, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestMiddleware_ClientIP(t *testing.T) {
	spoofed := map[string]string{"X-Forwarded-For": "203.0.113.7"}

	t.Run("forwarded header from untrusted client is ignored", func(t *testing.T) {
		l := &fakeLimiter{allowed: true}
		serve(t, l, policy, nil, spoofed)
		assert.Equal(t, []string{"auth:ip:10.0.0.5"}, l.keys)
	})

	t.Run("forwarded header from trusted proxy is used", func(t *testing.T) {
		l := &fakeLimiter{allowed: true}
		serve(t, l, policy, []string{"10.0.0.0/8"}, spoofed)
		assert.Equal(t, []string{"auth:ip:203.0.113.7"}, l.keys)
	})
}
