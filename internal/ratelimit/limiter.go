package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// slidingWindowLua implements a sliding window rate limiter in Lua.
// Key: rate limit key, ARGV[1]: window size in ms, ARGV[2]: max requests, ARGV[3]: current time in ms
var slidingWindowLua = redis.NewScript(`
local key = KEYS[1]
local window = tonumber(ARGV[1])
local max_requests = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

-- Count current entries
local count = redis.call('ZCARD', key)

if count < max_requests then
    redis.call('ZADD', key, now, now .. '-' .. math.random(1000000))
    redis.call('PEXPIRE', key, window)
    return 1
end

return 0
`)

type Limiter struct {
	client     *redis.Client
	maxPerSec  int
	windowMs   int64
}

func NewLimiter(client *redis.Client, maxPerSec int) *Limiter {
	return &Limiter{
		client:    client,
		maxPerSec: maxPerSec,
		windowMs:  1000,
	}
}

// Allow checks if a request for the given channel is within the rate limit.
func (l *Limiter) Allow(ctx context.Context, channel string) (bool, error) {
	key := fmt.Sprintf("ratelimit:%s", channel)
	now := time.Now().UnixMilli()

	result, err := slidingWindowLua.Run(ctx, l.client, []string{key}, l.windowMs, l.maxPerSec, now).Int()
	if err != nil {
		return false, fmt.Errorf("rate limit check: %w", err)
	}

	return result == 1, nil
}
