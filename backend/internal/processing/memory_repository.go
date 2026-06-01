package processing

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu   sync.RWMutex
	jobs map[string]Job
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{jobs: map[string]Job{}}
}

func (r *MemoryRepository) CreateJob(_ context.Context, job Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[job.ID] = job
	return nil
}

func (r *MemoryRepository) UpdateJobStatus(_ context.Context, jobID string, from, to JobStatus, updatedAt time.Time) (Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.jobs[jobID]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	if job.Status != from {
		return Job{}, ErrInvalidStatusTransition
	}

	job.Status = to
	job.UpdatedAt = updatedAt
	r.jobs[jobID] = job
	return job, nil
}

func (r *MemoryRepository) GetJob(_ context.Context, jobID string) (Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.jobs[jobID]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	return job, nil
}

func (r *MemoryRepository) ListJobsByProjectStatus(_ context.Context, projectID string, status JobStatus) ([]Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	jobs := make([]Job, 0)
	for _, job := range r.jobs {
		if job.ProjectID == projectID && job.Status == status {
			jobs = append(jobs, job)
		}
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].UpdatedAt.Equal(jobs[j].UpdatedAt) {
			return jobs[i].ID < jobs[j].ID
		}
		return jobs[i].UpdatedAt.Before(jobs[j].UpdatedAt)
	})
	return jobs, nil
}

func (r *MemoryRepository) DeleteJob(_ context.Context, jobID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.jobs, jobID)
	return nil
}
