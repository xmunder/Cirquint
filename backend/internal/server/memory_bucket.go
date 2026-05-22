package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/msi/circuit-storys/backend/internal/storage"
)

type memoryBucket struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func newMemoryBucket() *memoryBucket {
	return &memoryBucket{objects: map[string][]byte{}}
}

func (b *memoryBucket) Put(_ context.Context, key string, body io.Reader, _ storage.ObjectMeta) (storage.StoredObject, error) {
	payload, err := io.ReadAll(body)
	if err != nil {
		return storage.StoredObject{}, err
	}

	b.mu.Lock()
	b.objects[key] = payload
	b.mu.Unlock()

	return storage.StoredObject{Key: key, ETag: fmt.Sprintf("mem-%d", len(payload))}, nil
}

func (b *memoryBucket) Get(_ context.Context, key string) (io.ReadCloser, error) {
	b.mu.RLock()
	payload, ok := b.objects[key]
	b.mu.RUnlock()
	if !ok {
		return nil, storage.ErrObjectNotFound
	}
	return io.NopCloser(bytes.NewReader(payload)), nil
}

func (b *memoryBucket) Delete(_ context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.objects, key)
	return nil
}
