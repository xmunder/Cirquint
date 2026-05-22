package uploads

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrUploadNotFound = errors.New("upload not found")

type MemoryRepository struct {
	mu      sync.RWMutex
	uploads map[string]Upload
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{uploads: map[string]Upload{}}
}

func (r *MemoryRepository) CreateUpload(_ context.Context, upload Upload) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uploads[upload.ID] = upload
	return nil
}

func (r *MemoryRepository) UpdateUploadStatus(_ context.Context, uploadID, status string, updatedAt time.Time) (Upload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	upload, ok := r.uploads[uploadID]
	if !ok {
		return Upload{}, ErrUploadNotFound
	}
	upload.Status = status
	upload.UpdatedAt = updatedAt
	r.uploads[uploadID] = upload
	return upload, nil
}

func (r *MemoryRepository) GetUpload(_ context.Context, uploadID string) (Upload, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	upload, ok := r.uploads[uploadID]
	if !ok {
		return Upload{}, ErrUploadNotFound
	}
	return upload, nil
}

func (r *MemoryRepository) DeleteUpload(_ context.Context, uploadID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.uploads, uploadID)
	return nil
}
