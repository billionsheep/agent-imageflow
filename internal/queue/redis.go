package queue

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const listName = "queue:image_generation"

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
