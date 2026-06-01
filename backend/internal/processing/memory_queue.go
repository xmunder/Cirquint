package processing

import (
	"context"
	"sync"
)

type MemoryQueue struct {
	mu       sync.Mutex
	payloads []Payload
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{}
}

func (q *MemoryQueue) Enqueue(_ context.Context, payload Payload) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.payloads = append(q.payloads, payload)
	return nil
}

func (q *MemoryQueue) Dequeue(_ context.Context) (Payload, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.payloads) == 0 {
		return Payload{}, ErrQueueEmpty
	}
	payload := q.payloads[0]
	q.payloads = q.payloads[1:]
	return payload, nil
}
