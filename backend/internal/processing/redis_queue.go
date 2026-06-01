package processing

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	redis "github.com/redis/go-redis/v9"
)

const DefaultRedisQueueKey = "processing:jobs"

type redisClient interface {
	RPush(ctx context.Context, key string, values ...any) *redis.IntCmd
	LPop(ctx context.Context, key string) *redis.StringCmd
}

type RedisQueue struct {
	client redisClient
	key    string
}

func NewRedisQueue(addr, key string) *RedisQueue {
	return NewRedisQueueWithClient(redis.NewClient(&redis.Options{Addr: addr}), key)
}

func NewRedisQueueWithClient(client redisClient, key string) *RedisQueue {
	return &RedisQueue{client: client, key: normalizeRedisQueueKey(key)}
}

func (q *RedisQueue) Enqueue(ctx context.Context, payload Payload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.client.RPush(ctx, q.key, body).Err()
}

func (q *RedisQueue) Dequeue(ctx context.Context) (Payload, error) {
	raw, err := q.client.LPop(ctx, q.key).Result()
	if errors.Is(err, redis.Nil) {
		return Payload{}, ErrQueueEmpty
	}
	if err != nil {
		return Payload{}, err
	}

	var payload Payload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return Payload{}, err
	}
	return payload, nil
}

func normalizeRedisQueueKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return DefaultRedisQueueKey
	}
	return key
}
