package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const rateLimitKeyPrefix = "ratelimit:"

var slidingWindowScript = goredis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
local count = redis.call('ZCARD', key)

local allowed = 0
if count < limit then
	redis.call('ZADD', key, now, ARGV[4])
	redis.call('PEXPIRE', key, window)
	count = count + 1
	allowed = 1
end

local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
local reset = window
if oldest[2] then
	reset = tonumber(oldest[2]) + window - now
end

return {allowed, limit - count, reset}
`)

type RateLimiter struct {
	client *goredis.Client
	now    func() time.Time
}

func NewRateLimiter(client *goredis.Client) *RateLimiter {
	return &RateLimiter{client: client, now: time.Now}
}

func (l *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, remaining int, resetIn time.Duration, err error) {
	res, err := slidingWindowScript.Run(ctx, l.client,
		[]string{rateLimitKeyPrefix + key},
		l.now().UnixMilli(), window.Milliseconds(), limit, uuid.NewString(),
	).Int64Slice()
	if err != nil {
		return false, 0, 0, fmt.Errorf("rate limit %s: %w", key, err)
	}
	return res[0] == 1, int(max(res[1], 0)), time.Duration(max(res[2], 0)) * time.Millisecond, nil
}
