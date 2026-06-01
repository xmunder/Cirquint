package processing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/msi/circuit-storys/backend/internal/processing"
	redis "github.com/redis/go-redis/v9"
)

func TestRedisQueueRoundTrip(t *testing.T) {
	client := &fakeRedisClient{}
	queue := processing.NewRedisQueueWithClient(client, "jobs:test")
	payload := processing.Payload{JobID: "job-001", ProjectID: "prj-001", UploadID: "up-001", UploadVersion: 1}

	if err := queue.Enqueue(context.Background(), payload); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	got, err := queue.Dequeue(context.Background())
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got != payload {
		t.Fatalf("expected payload %+v, got %+v", payload, got)
	}
	if client.lastKey != "jobs:test" {
		t.Fatalf("expected queue key jobs:test, got %s", client.lastKey)
	}
	if _, err := queue.Dequeue(context.Background()); err != processing.ErrQueueEmpty {
		t.Fatalf("expected ErrQueueEmpty, got %v", err)
	}
}

func TestRedisQueueUsesDefaultKeyAndPropagatesErrors(t *testing.T) {
	client := &fakeRedisClient{popErr: errors.New("pop failed")}
	queue := processing.NewRedisQueueWithClient(client, "   ")
	if err := queue.Enqueue(context.Background(), processing.Payload{JobID: "job-001"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if client.lastKey != processing.DefaultRedisQueueKey {
		t.Fatalf("queue key = %q, want %q", client.lastKey, processing.DefaultRedisQueueKey)
	}
	if _, err := queue.Dequeue(context.Background()); !errors.Is(err, client.popErr) {
		t.Fatalf("dequeue error = %v, want %v", err, client.popErr)
	}

	client.values = []string{"not-json"}
	client.popErr = nil
	if _, err := queue.Dequeue(context.Background()); err == nil {
		t.Fatal("expected json decode error")
	}
}

type fakeRedisClient struct {
	lastKey string
	values  []string
	popErr  error
}

func (c *fakeRedisClient) RPush(_ context.Context, key string, values ...any) *redis.IntCmd {
	c.lastKey = key
	for _, value := range values {
		body, _ := value.([]byte)
		c.values = append(c.values, string(body))
	}
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(int64(len(c.values)))
	return cmd
}

func (c *fakeRedisClient) LPop(_ context.Context, key string) *redis.StringCmd {
	c.lastKey = key
	cmd := redis.NewStringCmd(context.Background())
	if c.popErr != nil {
		cmd.SetErr(c.popErr)
		return cmd
	}
	if len(c.values) == 0 {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	value := c.values[0]
	c.values = c.values[1:]
	cmd.SetVal(value)
	return cmd
}
