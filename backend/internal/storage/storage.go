package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

var ErrObjectNotFound = errors.New("object not found")

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

type Bucket interface {
	Put(ctx context.Context, key string, body io.Reader, meta ObjectMeta) (StoredObject, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

type R2Storage struct {
	bucket Bucket
}

func NewR2Storage(bucket Bucket) *R2Storage {
	return &R2Storage{bucket: bucket}
}

func (s *R2Storage) Put(ctx context.Context, key string, body io.Reader, meta ObjectMeta) (StoredObject, error) {
	return s.bucket.Put(ctx, key, body, meta)
}

func (s *R2Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.bucket.Get(ctx, key)
}

func UploadObjectKey(workspaceID, uploadID, filename string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	if ext == "jpeg" {
		ext = "jpg"
	}
	if ext == "" {
		ext = "bin"
	}

	return fmt.Sprintf("workspaces/%s/uploads/%s/v1/source.%s", workspaceID, uploadID, ext)
}
