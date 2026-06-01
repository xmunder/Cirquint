package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type filesystemBucket struct {
	root string
}

func NewFilesystemObjectStorage(root string) *R2Storage {
	return NewR2Storage(filesystemBucket{root: root})
}

func (b filesystemBucket) Put(_ context.Context, key string, body io.Reader, _ ObjectMeta) (StoredObject, error) {
	path := filepath.Join(b.root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return StoredObject{}, err
	}

	file, err := os.Create(path)
	if err != nil {
		return StoredObject{}, err
	}
	defer file.Close()

	size, err := io.Copy(file, body)
	if err != nil {
		return StoredObject{}, err
	}

	return StoredObject{Key: key, ETag: fmt.Sprintf("fs-%d", size)}, nil
}

func (b filesystemBucket) Get(_ context.Context, key string) (io.ReadCloser, error) {
	file, err := os.Open(filepath.Join(b.root, filepath.FromSlash(key)))
	if os.IsNotExist(err) {
		return nil, ErrObjectNotFound
	}
	return file, err
}

func (b filesystemBucket) Delete(_ context.Context, key string) error {
	err := os.Remove(filepath.Join(b.root, filepath.FromSlash(key)))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
