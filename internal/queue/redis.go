package queue

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	listName          = "queue:image_generation"
	scheduledListName = "queue:image_generation:scheduled"
)

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue(redisURL string) (*RedisQueue, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &RedisQueue{client: redis.NewClient(opts)}, nil
}

func (q *RedisQueue) Close() error {
	return q.client.Close()
}

func (q *RedisQueue) Ping(ctx context.Context) error {
	return q.client.Ping(ctx).Err()
}

func (q *RedisQueue) Enqueue(ctx context.Context, taskID string) error {
	return q.client.RPush(ctx, listName, taskID).Err()
}

func (q *RedisQueue) EnqueueAfter(ctx context.Context, taskID string, delay time.Duration) error {
	if delay <= 0 {
		return q.Enqueue(ctx, taskID)
	}
	readyAt := time.Now().UTC().Add(delay).UnixMilli()
	return q.client.ZAdd(ctx, scheduledListName, redis.Z{
		Score:  float64(readyAt),
		Member: taskID,
	}).Err()
}

func (q *RedisQueue) PromoteScheduled(ctx context.Context, limit int) (int, error) {
	if limit < 1 {
		limit = 1
	}
	result, err := q.client.Eval(ctx, `
local scheduled = KEYS[1]
local queue = KEYS[2]
local now = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local promoted = redis.call("ZRANGEBYSCORE", scheduled, "-inf", now, "LIMIT", 0, limit)
if #promoted == 0 then
	return 0
end
for _, taskId in ipairs(promoted) do
	redis.call("ZREM", scheduled, taskId)
	redis.call("RPUSH", queue, taskId)
end
return #promoted
`, []string{scheduledListName, listName}, time.Now().UTC().UnixMilli(), limit).Result()
	if err != nil {
		return 0, err
	}
	switch value := result.(type) {
	case int64:
		return int(value), nil
	case string:
		parsed, parseErr := strconv.Atoi(value)
		if parseErr != nil {
			return 0, parseErr
		}
		return parsed, nil
	default:
		return 0, nil
	}
}

func (q *RedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := q.client.BLPop(ctx, timeout, listName).Result()
	if err != nil {
		return "", err
	}
	if len(result) < 2 {
		return "", nil
	}
	return result[1], nil
}

func (q *RedisQueue) LockTask(ctx context.Context, taskID string, ttl time.Duration) (bool, error) {
	return q.client.SetNX(ctx, "task:"+taskID+":lock", "1", ttl).Result()
}

func (q *RedisQueue) UnlockTask(ctx context.Context, taskID string) {
	_ = q.client.Del(ctx, "task:"+taskID+":lock").Err()
}
