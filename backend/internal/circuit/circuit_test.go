package circuit_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/server"
)

func TestDecideStatusByPolicy(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		warnings   []string
		want       string
	}{
		{name: "ready when confidence meets threshold", confidence: 0.91, want: circuit.StatusReady},
		{name: "needs review below threshold", confidence: 0.79, want: circuit.StatusNeedsReview},
		{name: "needs review on ambiguous warning", confidence: 0.99, warnings: []string{"ambiguous-node-label"}, want: circuit.StatusNeedsReview},
		{name: "needs review on incomplete circuit warning", confidence: 0.95, warnings: []string{"incomplete-circuit:missing-ground"}, want: circuit.StatusNeedsReview},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := circuit.DecideStatus(tt.confidence, tt.warnings, 0.8)
			if got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestPersistIsIdempotentForJob(t *testing.T) {
	storage := server.NewMemoryObjectStorage()
	service := circuit.NewService(
		circuit.NewMemoryRepository(),
		storage,
		fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)},
		&sequenceIDs{ids: []string{"ext-001", "cir-001", "ext-should-not-be-used", "cir-should-not-be-used"}},
		0.8,
	)
	job := processing.Job{ID: "job-001", WorkspaceID: "ws-001", ProjectID: "prj-001", UploadID: "up-001"}
	result := circuit.ExtractionResult{Provider: "mock", Confidence: 0.92, Warnings: []string{"non-blocking"}}

	first, err := service.Persist(context.Background(), circuit.PersistInput{
		WorkspaceID: "ws-001",
		ProjectID:   "prj-001",
		UploadID:    "up-001",
		Job:         job,
		Result:      result,
	})
	if err != nil {
		t.Fatalf("first persist: %v", err)
	}

	second, err := service.Persist(context.Background(), circuit.PersistInput{
		WorkspaceID: "ws-001",
		ProjectID:   "prj-001",
		UploadID:    "up-001",
		Job:         job,
		Result:      circuit.ExtractionResult{Provider: "other", Confidence: 0.1, Warnings: []string{"ambiguous"}},
	})
	if err != nil {
		t.Fatalf("second persist: %v", err)
	}

	if second.Extraction.ID != first.Extraction.ID {
		t.Fatalf("expected same extraction revision, got %s and %s", first.Extraction.ID, second.Extraction.ID)
	}
	if second.Circuit.ID != first.Circuit.ID {
		t.Fatalf("expected same circuit revision, got %s and %s", first.Circuit.ID, second.Circuit.ID)
	}
	if second.Status != processing.JobStatusReady {
		t.Fatalf("expected ready status, got %s", second.Status)
	}

	body, err := storage.Get(context.Background(), first.Circuit.ObjectKey)
	if err != nil {
		t.Fatalf("get stored circuit spec: %v", err)
	}
	defer body.Close()

	var spec circuit.CircuitSpec
	if err := json.NewDecoder(body).Decode(&spec); err != nil {
		t.Fatalf("decode stored circuit spec: %v", err)
	}
	if spec.Status != circuit.StatusReady {
		t.Fatalf("expected stored ready status, got %s", spec.Status)
	}
	if spec.Confidence != 0.92 {
		t.Fatalf("expected original confidence, got %v", spec.Confidence)
	}

	extractionBody, err := storage.Get(context.Background(), first.Extraction.ObjectKey)
	if err != nil {
		t.Fatalf("get stored extraction revision: %v", err)
	}
	defer extractionBody.Close()

	var extraction circuit.ExtractionRevision
	if err := json.NewDecoder(extractionBody).Decode(&extraction); err != nil {
		t.Fatalf("decode stored extraction revision: %v", err)
	}
	if !reflect.DeepEqual(extraction.Result, result) {
		t.Fatalf("expected stored extraction result %+v, got %+v", result, extraction.Result)
	}
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type sequenceIDs struct {
	ids []string
	idx int
}

func (g *sequenceIDs) NewID() string {
	if g.idx >= len(g.ids) {
		return "fallback-id"
	}
	id := g.ids[g.idx]
	g.idx++
	return id
}
