package processing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/processing"
)

func TestJobStateTransitions(t *testing.T) {
	repo := processing.NewMemoryRepository()
	svc := processing.NewService(repo, fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}, fixedIDGenerator{id: "job-001"})

	job, err := svc.CreateJob(context.Background(), "ws-001", "prj-001", "up-001", 1)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if job.Status != processing.JobStatusCreated {
		t.Fatalf("expected created status, got %s", job.Status)
	}

	job, err = svc.Transition(context.Background(), job.ID, processing.JobStatusCreated, processing.JobStatusUploaded)
	if err != nil {
		t.Fatalf("transition to uploaded: %v", err)
	}

	job, err = svc.Transition(context.Background(), job.ID, processing.JobStatusUploaded, processing.JobStatusQueued)
	if err != nil {
		t.Fatalf("transition to queued: %v", err)
	}
	if job.Status != processing.JobStatusQueued {
		t.Fatalf("expected queued status, got %s", job.Status)
	}

	if _, err := svc.Transition(context.Background(), job.ID, processing.JobStatusCreated, processing.JobStatusQueued); err == nil {
		t.Fatal("expected invalid transition error")
	}
}

func TestClaimQueuedUsesCompareAndSwap(t *testing.T) {
	repo := processing.NewMemoryRepository()
	queue := processing.NewMemoryQueue()
	svc := processing.NewService(repo, fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}, fixedIDGenerator{id: "job-001"}).WithQueue(queue)

	job, err := svc.CreateJob(context.Background(), "ws-001", "prj-001", "up-001", 1)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	job, err = svc.Transition(context.Background(), job.ID, processing.JobStatusCreated, processing.JobStatusUploaded)
	if err != nil {
		t.Fatalf("transition to uploaded: %v", err)
	}
	job, err = svc.Transition(context.Background(), job.ID, processing.JobStatusUploaded, processing.JobStatusQueued)
	if err != nil {
		t.Fatalf("transition to queued: %v", err)
	}
	if err := svc.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue job: %v", err)
	}

	payload, err := svc.Dequeue(context.Background())
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	claimed, err := svc.ClaimQueued(context.Background(), payload)
	if err != nil {
		t.Fatalf("claim queued: %v", err)
	}
	if claimed.Status != processing.JobStatusExtracting {
		t.Fatalf("expected extracting status, got %s", claimed.Status)
	}
	if _, err := svc.ClaimQueued(context.Background(), payload); err == nil {
		t.Fatal("expected compare-and-swap failure on duplicate claim")
	}
}

func TestQueueOptionalHelpers(t *testing.T) {
	repo := processing.NewMemoryRepository()
	svc := processing.NewService(repo, fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}, fixedIDGenerator{id: "job-001"})

	job, err := svc.CreateJob(context.Background(), "ws-001", "prj-001", "up-001", 1)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := svc.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue without queue error = %v", err)
	}
	if _, err := svc.Dequeue(context.Background()); err != processing.ErrQueueEmpty {
		t.Fatalf("dequeue error = %v, want %v", err, processing.ErrQueueEmpty)
	}

	job, err = svc.Transition(context.Background(), job.ID, processing.JobStatusCreated, processing.JobStatusExtracting)
	if err != nil {
		t.Fatalf("transition to extracting: %v", err)
	}

	job, err = svc.MarkCompleted(context.Background(), job.ID, processing.JobStatusReady)
	if err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	if job.Status != processing.JobStatusReady {
		t.Fatalf("status = %s, want %s", job.Status, processing.JobStatusReady)
	}

	job, err = svc.Transition(context.Background(), job.ID, processing.JobStatusReady, processing.JobStatusExtracting)
	if err != nil {
		t.Fatalf("transition to extracting: %v", err)
	}
	job, err = svc.MarkFailed(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if job.Status != processing.JobStatusFailed {
		t.Fatalf("status = %s, want %s", job.Status, processing.JobStatusFailed)
	}

	if _, err := svc.GetJob(context.Background(), job.ID); err != nil {
		t.Fatalf("get job: %v", err)
	}
	if err := svc.DeleteJob(context.Background(), job.ID); err != nil {
		t.Fatalf("delete job: %v", err)
	}
	if _, err := svc.GetJob(context.Background(), job.ID); err != processing.ErrJobNotFound {
		t.Fatalf("get deleted job error = %v, want %v", err, processing.ErrJobNotFound)
	}
}

func TestResolveReviewTransitionsNeedsReviewToReadyAndFailed(t *testing.T) {
	t.Run("ready", func(t *testing.T) {
		repo := processing.NewMemoryRepository()
		svc := processing.NewService(repo, fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}, fixedIDGenerator{id: "job-001"})
		job := createNeedsReviewJob(t, svc)

		resolved, err := svc.ResolveReview(context.Background(), job.ID, processing.JobStatusReady)
		if err != nil {
			t.Fatalf("ResolveReview error = %v", err)
		}
		if resolved.Status != processing.JobStatusReady {
			t.Fatalf("resolved status = %s, want %s", resolved.Status, processing.JobStatusReady)
		}
	})

	t.Run("failed", func(t *testing.T) {
		repo := processing.NewMemoryRepository()
		svc := processing.NewService(repo, fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}, fixedIDGenerator{id: "job-001"})
		job := createNeedsReviewJob(t, svc)

		resolved, err := svc.ResolveReview(context.Background(), job.ID, processing.JobStatusFailed)
		if err != nil {
			t.Fatalf("ResolveReview error = %v", err)
		}
		if resolved.Status != processing.JobStatusFailed {
			t.Fatalf("resolved status = %s, want %s", resolved.Status, processing.JobStatusFailed)
		}
	})
}

func TestResolveReviewRejectsInvalidTarget(t *testing.T) {
	repo := processing.NewMemoryRepository()
	svc := processing.NewService(repo, fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}, fixedIDGenerator{id: "job-001"})
	job := createNeedsReviewJob(t, svc)

	_, err := svc.ResolveReview(context.Background(), job.ID, processing.JobStatusQueued)
	if !errors.Is(err, processing.ErrInvalidStatusTransition) {
		t.Fatalf("ResolveReview error = %v, want %v", err, processing.ErrInvalidStatusTransition)
	}
}

func TestListJobsByProjectStatusFiltersByProjectAndStatus(t *testing.T) {
	repo := processing.NewMemoryRepository()
	svc := processing.NewService(repo, fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}, fixedIDGenerator{id: "job-001"})
	ready := createNeedsReviewJob(t, svc)
	if _, err := svc.ResolveReview(context.Background(), ready.ID, processing.JobStatusReady); err != nil {
		t.Fatalf("ResolveReview ready error = %v", err)
	}
	createNeedsReviewJobWithID(t, repo, "job-002", "prj-001", processing.JobStatusNeedsReview)
	createNeedsReviewJobWithID(t, repo, "job-003", "prj-002", processing.JobStatusNeedsReview)

	jobs, err := svc.ListByProjectStatus(context.Background(), "prj-001", processing.JobStatusNeedsReview)
	if err != nil {
		t.Fatalf("ListByProjectStatus error = %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1", len(jobs))
	}
	if jobs[0].ID != "job-002" {
		t.Fatalf("jobs[0].ID = %s, want job-002", jobs[0].ID)
	}
}

func createNeedsReviewJob(t *testing.T, svc *processing.Service) processing.Job {
	t.Helper()
	job, err := svc.CreateJob(context.Background(), "ws-001", "prj-001", "up-001", 1)
	if err != nil {
		t.Fatalf("CreateJob error = %v", err)
	}
	job, err = svc.Transition(context.Background(), job.ID, processing.JobStatusCreated, processing.JobStatusExtracting)
	if err != nil {
		t.Fatalf("transition to extracting: %v", err)
	}
	job, err = svc.MarkCompleted(context.Background(), job.ID, processing.JobStatusNeedsReview)
	if err != nil {
		t.Fatalf("MarkCompleted needs_review error = %v", err)
	}
	return job
}

func createNeedsReviewJobWithID(t *testing.T, repo *processing.MemoryRepository, jobID, projectID string, status processing.JobStatus) {
	t.Helper()
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	if err := repo.CreateJob(context.Background(), processing.Job{ID: jobID, WorkspaceID: "ws-001", ProjectID: projectID, UploadID: "up-001", Status: status, Payload: processing.Payload{JobID: jobID, ProjectID: projectID, UploadID: "up-001", UploadVersion: 1}, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("CreateJob error = %v", err)
	}
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fixedIDGenerator struct{ id string }

func (g fixedIDGenerator) NewID() string { return g.id }
