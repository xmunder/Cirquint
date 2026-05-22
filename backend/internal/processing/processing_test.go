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

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fixedIDGenerator struct{ id string }

func (g fixedIDGenerator) NewID() string { return g.id }
