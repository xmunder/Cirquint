package circuit

import (
	"context"
	"sync"

	"github.com/msi/circuit-storys/backend/internal/processing"
)

type MemoryRepository struct {
	mu        sync.RWMutex
	pairs     map[string]RevisionPair
	byProject map[string]CircuitRevision
	decisions map[string]ReviewDecisionRecord
	jobs      processing.Repository
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{pairs: map[string]RevisionPair{}, byProject: map[string]CircuitRevision{}, decisions: map[string]ReviewDecisionRecord{}}
}

func (r *MemoryRepository) WithProcessingRepository(repo processing.Repository) *MemoryRepository {
	r.jobs = repo
	return r
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

func (r *MemoryRepository) GetLatestReviewable(_ context.Context, projectID, jobID string) (RevisionPair, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pair, ok := r.pairs[jobID]
	if !ok || pair.Circuit.ProjectID != projectID || pair.Circuit.Status != StatusNeedsReview {
		return RevisionPair{}, false, nil
	}
	return cloneRevisionPair(pair), true, nil
}

func (r *MemoryRepository) ListLatestReviewable(_ context.Context, projectID string) ([]RevisionPair, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pairs := make([]RevisionPair, 0)
	for _, pair := range r.pairs {
		if pair.Circuit.ProjectID == projectID && pair.Circuit.Status == StatusNeedsReview {
			pairs = append(pairs, cloneRevisionPair(pair))
		}
	}
	return pairs, nil
}

func (r *MemoryRepository) SaveReviewDecision(ctx context.Context, decision ReviewDecisionRecord, resolved CircuitRevision) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pair, ok := r.pairs[decision.JobID]
	if !ok || pair.Circuit.ProjectID != decision.ProjectID || pair.Circuit.ID != decision.ReviewedCircuitRevisionID || pair.Circuit.Status != StatusNeedsReview {
		return ErrReviewNotFound
	}
	if _, exists := r.decisions[decision.ReviewedCircuitRevisionID]; exists {
		return ErrRevisionAlreadyReviewed
	}
	if r.jobs != nil {
		if _, err := r.jobs.UpdateJobStatus(ctx, decision.JobID, processing.JobStatusNeedsReview, statusFromCircuit(resolved.Status), resolved.UpdatedAt); err != nil {
			return err
		}
	}
	pair.Circuit = cloneCircuitRevision(resolved)
	r.pairs[decision.JobID] = pair
	r.byProject[pair.Circuit.ProjectID] = pair.Circuit
	r.decisions[decision.ReviewedCircuitRevisionID] = decision
	return nil
}

func (r *MemoryRepository) GetReviewDecision(_ context.Context, reviewedCircuitRevisionID string) (ReviewDecisionRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decision, ok := r.decisions[reviewedCircuitRevisionID]
	return decision, ok, nil
}

func cloneRevisionPair(pair RevisionPair) RevisionPair {
	return RevisionPair{Extraction: cloneExtractionRevision(pair.Extraction), Circuit: cloneCircuitRevision(pair.Circuit)}
}
