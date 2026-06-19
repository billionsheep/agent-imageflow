package httpapi

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string, maxRequests int, window time.Duration) (RateLimitDecision, error)
	Close() error
}

type RateLimitDecision struct {
	Allowed    bool
	Count      int
	RetryAfter time.Duration
}

type RedisRateLimiter struct {
	client *redis.Client
}

const rateLimitIncrementScript = `
local current = redis.call("INCR", KEYS[1])
if current == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return current
`

func NewRedisRateLimiter(redisURL string) (*RedisRateLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &RedisRateLimiter{client: redis.NewClient(opts)}, nil
}

func (l *RedisRateLimiter) Close() error {
	if l == nil || l.client == nil {
		return nil
	}
	return l.client.Close()
}

func (l *RedisRateLimiter) Allow(ctx context.Context, key string, maxRequests int, window time.Duration) (RateLimitDecision, error) {
	if l == nil || l.client == nil || maxRequests < 1 || window <= 0 {
		return RateLimitDecision{Allowed: true}, nil
	}

	now := time.Now().UTC()
	windowStart := now.Truncate(window)
	windowEnd := windowStart.Add(window)
	retryAfter := windowEnd.Sub(now)
	if retryAfter <= 0 {
		retryAfter = time.Second
	}

	bucketKey := fmt.Sprintf("%s:%s", key, strconv.FormatInt(windowStart.UnixMilli(), 10))
	count, err := l.client.Eval(ctx, rateLimitIncrementScript, []string{bucketKey}, window.Milliseconds()).Int64()
	if err != nil {
		return RateLimitDecision{}, err
	}

	return RateLimitDecision{
		Allowed:    count <= int64(maxRequests),
		Count:      int(count),
		RetryAfter: retryAfter,
	}, nil
}
