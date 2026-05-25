package circuit

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu        sync.RWMutex
	pairs     map[string]RevisionPair
	byProject map[string]CircuitRevision
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{pairs: map[string]RevisionPair{}, byProject: map[string]CircuitRevision{}}
}

func (r *MemoryRepository) GetByJobID(_ context.Context, jobID string) (RevisionPair, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pair, ok := r.pairs[jobID]
	return pair, ok, nil
}

func (r *MemoryRepository) Save(_ context.Context, pair RevisionPair) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.pairs[pair.Extraction.JobID]; ok {
		return nil
	}
	r.pairs[pair.Extraction.JobID] = pair
	r.byProject[pair.Circuit.ProjectID] = pair.Circuit
	return nil
}
