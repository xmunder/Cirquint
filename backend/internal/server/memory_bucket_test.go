package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/msi/circuit-storys/backend/internal/storage"
)

func TestMemoryBucketRoundTripAndDelete(t *testing.T) {
	bucket := newMemoryBucket()
	stored, err := bucket.Put(context.Background(), "key-1", bytes.NewBufferString("payload"), storage.ObjectMeta{})
	if err != nil {
		t.Fatalf("Put error = %v", err)
	}
	if stored.Key != "key-1" {
		t.Fatalf("stored key = %q, want key-1", stored.Key)
	}

	body, err := bucket.Get(context.Background(), "key-1")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	defer body.Close()
	payload, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll error = %v", err)
	}
	if string(payload) != "payload" {
		t.Fatalf("payload = %q, want payload", string(payload))
	}

	if err := bucket.Delete(context.Background(), "key-1"); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	_, err = bucket.Get(context.Background(), "key-1")
	if !errors.Is(err, storage.ErrObjectNotFound) {
		t.Fatalf("Get after delete error = %v, want %v", err, storage.ErrObjectNotFound)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestMemoryBucketPutPropagatesReadError(t *testing.T) {
	bucket := newMemoryBucket()
	_, err := bucket.Put(context.Background(), "key-1", errReader{}, storage.ObjectMeta{})
	if err == nil {
		t.Fatal("expected read error")
	}
}
