package storage_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/msi/circuit-storys/backend/internal/storage"
)

func TestUploadObjectKey(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "png", filename: "diagram.png", want: "workspaces/ws-001/uploads/up-001/v1/source.png"},
		{name: "jpeg normalized", filename: "diagram.jpeg", want: "workspaces/ws-001/uploads/up-001/v1/source.jpg"},
		{name: "missing ext falls back", filename: "diagram", want: "workspaces/ws-001/uploads/up-001/v1/source.bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := storage.UploadObjectKey("ws-001", "up-001", tt.filename)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

type bucketSpy struct {
	putKey    string
	getKey    string
	deleteKey string
}

func (b *bucketSpy) Put(_ context.Context, key string, body io.Reader, _ storage.ObjectMeta) (storage.StoredObject, error) {
	b.putKey = key
	_, _ = io.ReadAll(body)
	return storage.StoredObject{Key: key, ETag: "etag"}, nil
}

func (b *bucketSpy) Get(_ context.Context, key string) (io.ReadCloser, error) {
	b.getKey = key
	return io.NopCloser(bytes.NewBufferString("payload")), nil
}

func (b *bucketSpy) Delete(_ context.Context, key string) error {
	b.deleteKey = key
	return nil
}

func TestR2StorageDelegatesToBucket(t *testing.T) {
	bucket := &bucketSpy{}
	store := storage.NewR2Storage(bucket)

	stored, err := store.Put(context.Background(), "key-1", bytes.NewBufferString("payload"), storage.ObjectMeta{})
	if err != nil {
		t.Fatalf("Put error = %v", err)
	}
	if stored.Key != "key-1" || bucket.putKey != "key-1" {
		t.Fatalf("Put delegation mismatch: stored=%q bucket=%q", stored.Key, bucket.putKey)
	}

	body, err := store.Get(context.Background(), "key-1")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	defer body.Close()
	if bucket.getKey != "key-1" {
		t.Fatalf("Get key = %q, want key-1", bucket.getKey)
	}

	if err := store.Delete(context.Background(), "key-1"); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if bucket.deleteKey != "key-1" {
		t.Fatalf("Delete key = %q, want key-1", bucket.deleteKey)
	}
}
