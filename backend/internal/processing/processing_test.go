package processing_test

import (
	"context"
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

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fixedIDGenerator struct{ id string }

func (g fixedIDGenerator) NewID() string { return g.id }
