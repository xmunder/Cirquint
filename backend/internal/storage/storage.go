package storage

import (
	"context"
	"io"
)

type ObjectMeta struct {
	ContentType string
	SizeBytes   int64
}

type StoredObject struct {
	Key  string
	ETag string
}

type ObjectStorage interface {
	Put(ctx context.Context, key string, body io.Reader, meta ObjectMeta) (StoredObject, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}
