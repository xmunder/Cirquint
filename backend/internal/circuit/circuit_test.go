package circuit_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/server"
	"github.com/msi/circuit-storys/backend/internal/storage"
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

func TestListReviewQueueFiltersByProjectAndReviewState(t *testing.T) {
	env := newReviewEnv(t)
	queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})
	env.seedJob(t, "prj-001", 0.98, nil)
	env.seedJob(t, "prj-002", 0.4, []string{"ambiguous-node-label"})

	items, err := env.circuits.ListReviewQueue(context.Background(), "prj-001")
	if err != nil {
		t.Fatalf("ListReviewQueue error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].JobID != queued.job.ID {
		t.Fatalf("item job_id = %s, want %s", items[0].JobID, queued.job.ID)
	}
	if items[0].CircuitRevisionID != queued.persisted.Circuit.ID {
		t.Fatalf("item circuit_revision_id = %s, want %s", items[0].CircuitRevisionID, queued.persisted.Circuit.ID)
	}
	if items[0].UploadID != queued.job.UploadID {
		t.Fatalf("item upload_id = %s, want %s", items[0].UploadID, queued.job.UploadID)
	}
	if items[0].Confidence != queued.persisted.Circuit.Confidence {
		t.Fatalf("item confidence = %v, want %v", items[0].Confidence, queued.persisted.Circuit.Confidence)
	}
	if !reflect.DeepEqual(items[0].Warnings, queued.persisted.Circuit.Warnings) {
		t.Fatalf("item warnings = %+v, want %+v", items[0].Warnings, queued.persisted.Circuit.Warnings)
	}
	if !items[0].QueuedForReviewAt.Equal(queued.persisted.Circuit.CreatedAt) {
		t.Fatalf("item queued_for_review_at = %s, want %s", items[0].QueuedForReviewAt, queued.persisted.Circuit.CreatedAt)
	}
}

func TestGetReviewDetailReturnsReviewableArtifacts(t *testing.T) {
	env := newReviewEnv(t)
	queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})

	detail, err := env.circuits.GetReviewDetail(context.Background(), "prj-001", queued.job.ID)
	if err != nil {
		t.Fatalf("GetReviewDetail error = %v", err)
	}
	if detail.Job.ID != queued.job.ID {
		t.Fatalf("detail job_id = %s, want %s", detail.Job.ID, queued.job.ID)
	}
	if detail.Extraction.ID != queued.persisted.Extraction.ID {
		t.Fatalf("detail extraction_id = %s, want %s", detail.Extraction.ID, queued.persisted.Extraction.ID)
	}
	if !reflect.DeepEqual(detail.Circuit, queued.persisted.Circuit.Spec) {
		t.Fatalf("detail circuit = %+v, want %+v", detail.Circuit, queued.persisted.Circuit.Spec)
	}
	if detail.ReviewState.JobStatus != processing.JobStatusNeedsReview {
		t.Fatalf("detail job status = %s, want %s", detail.ReviewState.JobStatus, processing.JobStatusNeedsReview)
	}
	if detail.ReviewState.CircuitRevisionID != queued.persisted.Circuit.ID {
		t.Fatalf("detail circuit_revision_id = %s, want %s", detail.ReviewState.CircuitRevisionID, queued.persisted.Circuit.ID)
	}
	if detail.ReviewState.CircuitStatus != circuit.StatusNeedsReview {
		t.Fatalf("detail circuit status = %s, want %s", detail.ReviewState.CircuitStatus, circuit.StatusNeedsReview)
	}

	if _, err := env.circuits.GetReviewDetail(context.Background(), "prj-002", queued.job.ID); !errors.Is(err, circuit.ErrReviewNotFound) {
		t.Fatalf("cross-project GetReviewDetail error = %v, want %v", err, circuit.ErrReviewNotFound)
	}
}

func TestSubmitReviewDecisionRejectsStaleRevision(t *testing.T) {
	env := newReviewEnv(t)
	queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})

	_, err := env.circuits.SubmitReviewDecision(context.Background(), circuit.SubmitReviewDecisionInput{
		ProjectID:            "prj-001",
		JobID:                queued.job.ID,
		CircuitRevisionID:    "cir-stale",
		Decision:             circuit.ReviewDecisionApprove,
		Note:                 "looks good",
		ReviewerID:           "reviewer-001",
		ReviewedAt:           time.Date(2026, 5, 23, 9, 0, 0, 0, time.UTC),
		CorrectedCircuitSpec: &circuit.CircuitSpec{Confidence: 0.99},
	})
	if !errors.Is(err, circuit.ErrStaleReviewRevision) {
		t.Fatalf("SubmitReviewDecision error = %v, want %v", err, circuit.ErrStaleReviewRevision)
	}

	items, err := env.circuits.ListReviewQueue(context.Background(), "prj-001")
	if err != nil {
		t.Fatalf("ListReviewQueue error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) after stale decision = %d, want 1", len(items))
	}
}

func TestSubmitReviewDecisionApproveWithCorrectionCreatesSuccessorAndAudit(t *testing.T) {
	env := newReviewEnv(t)
	queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})
	corrected := circuit.CircuitSpec{Confidence: 0.99, Warnings: []string{"human-corrected"}}

	result, err := env.circuits.SubmitReviewDecision(context.Background(), circuit.SubmitReviewDecisionInput{
		ProjectID:            "prj-001",
		JobID:                queued.job.ID,
		CircuitRevisionID:    queued.persisted.Circuit.ID,
		Decision:             circuit.ReviewDecisionApprove,
		Note:                 "fixed labels",
		ReviewerID:           "reviewer-001",
		ReviewedAt:           time.Date(2026, 5, 23, 9, 0, 0, 0, time.UTC),
		CorrectedCircuitSpec: &corrected,
	})
	if err != nil {
		t.Fatalf("SubmitReviewDecision error = %v", err)
	}
	if result.ResolvedRevision.ID == queued.persisted.Circuit.ID {
		t.Fatal("expected successor revision for corrected approve")
	}
	if result.ResolvedRevision.Version != queued.persisted.Circuit.Version+1 {
		t.Fatalf("resolved version = %d, want %d", result.ResolvedRevision.Version, queued.persisted.Circuit.Version+1)
	}
	if result.ResolvedRevision.Status != circuit.StatusReady {
		t.Fatalf("resolved status = %s, want %s", result.ResolvedRevision.Status, circuit.StatusReady)
	}
	if result.Decision.ReviewerID != "reviewer-001" {
		t.Fatalf("decision reviewer_id = %s, want reviewer-001", result.Decision.ReviewerID)
	}

	pair, ok, err := env.repo.GetByJobID(context.Background(), queued.job.ID)
	if err != nil {
		t.Fatalf("GetByJobID error = %v", err)
	}
	if !ok {
		t.Fatal("expected stored pair")
	}
	if pair.Circuit.ID != result.ResolvedRevision.ID {
		t.Fatalf("latest circuit revision = %s, want %s", pair.Circuit.ID, result.ResolvedRevision.ID)
	}

	job, err := env.jobs.GetJob(context.Background(), queued.job.ID)
	if err != nil {
		t.Fatalf("GetJob error = %v", err)
	}
	if job.Status != processing.JobStatusReady {
		t.Fatalf("job status = %s, want %s", job.Status, processing.JobStatusReady)
	}

	stored := readStoredCircuitSpec(t, env.storage, result.ResolvedRevision.ObjectKey)
	if stored.Status != circuit.StatusReady {
		t.Fatalf("stored corrected status = %s, want %s", stored.Status, circuit.StatusReady)
	}
	if !reflect.DeepEqual(stored.Warnings, corrected.Warnings) {
		t.Fatalf("stored corrected warnings = %+v, want %+v", stored.Warnings, corrected.Warnings)
	}

	audit, ok, err := env.repo.GetReviewDecision(context.Background(), queued.persisted.Circuit.ID)
	if err != nil {
		t.Fatalf("GetReviewDecision error = %v", err)
	}
	if !ok {
		t.Fatal("expected stored review audit")
	}
	if audit.ResolvedCircuitRevisionID != result.ResolvedRevision.ID {
		t.Fatalf("audit resolved_circuit_revision_id = %s, want %s", audit.ResolvedCircuitRevisionID, result.ResolvedRevision.ID)
	}
}

func TestSubmitReviewDecisionRejectWithoutCorrectionMarksFailed(t *testing.T) {
	env := newReviewEnv(t)
	queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})

	result, err := env.circuits.SubmitReviewDecision(context.Background(), circuit.SubmitReviewDecisionInput{
		ProjectID:         "prj-001",
		JobID:             queued.job.ID,
		CircuitRevisionID: queued.persisted.Circuit.ID,
		Decision:          circuit.ReviewDecisionReject,
		Note:              "not salvageable",
		ReviewerID:        "reviewer-002",
		ReviewedAt:        time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("SubmitReviewDecision error = %v", err)
	}
	if result.ResolvedRevision.ID != queued.persisted.Circuit.ID {
		t.Fatalf("resolved revision id = %s, want %s", result.ResolvedRevision.ID, queued.persisted.Circuit.ID)
	}
	if result.ResolvedRevision.Status != circuit.StatusFailed {
		t.Fatalf("resolved status = %s, want %s", result.ResolvedRevision.Status, circuit.StatusFailed)
	}

	job, err := env.jobs.GetJob(context.Background(), queued.job.ID)
	if err != nil {
		t.Fatalf("GetJob error = %v", err)
	}
	if job.Status != processing.JobStatusFailed {
		t.Fatalf("job status = %s, want %s", job.Status, processing.JobStatusFailed)
	}

	items, err := env.circuits.ListReviewQueue(context.Background(), "prj-001")
	if err != nil {
		t.Fatalf("ListReviewQueue error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) after rejection = %d, want 0", len(items))
	}
}

func TestSubmitReviewDecisionAllowsOnlyOneDecisionPerRevision(t *testing.T) {
	env := newReviewEnv(t)
	queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})

	_, err := env.circuits.SubmitReviewDecision(context.Background(), circuit.SubmitReviewDecisionInput{
		ProjectID:         "prj-001",
		JobID:             queued.job.ID,
		CircuitRevisionID: queued.persisted.Circuit.ID,
		Decision:          circuit.ReviewDecisionReject,
		Note:              "first decision",
		ReviewerID:        "reviewer-001",
		ReviewedAt:        time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("first SubmitReviewDecision error = %v", err)
	}

	_, err = env.circuits.SubmitReviewDecision(context.Background(), circuit.SubmitReviewDecisionInput{
		ProjectID:         "prj-001",
		JobID:             queued.job.ID,
		CircuitRevisionID: queued.persisted.Circuit.ID,
		Decision:          circuit.ReviewDecisionApprove,
		Note:              "second decision",
		ReviewerID:        "reviewer-002",
		ReviewedAt:        time.Date(2026, 5, 23, 11, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, circuit.ErrRevisionAlreadyReviewed) {
		t.Fatalf("second SubmitReviewDecision error = %v, want %v", err, circuit.ErrRevisionAlreadyReviewed)
	}
}

func TestSubmitReviewDecisionRollsBackWhenJobTransitionFails(t *testing.T) {
	repo := circuit.NewMemoryRepository()
	clock := fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}
	ids := &sequenceIDs{ids: []string{"decision-001"}}
	storage := server.NewMemoryObjectStorage()
	circuits := circuit.NewService(repo.WithProcessingRepository(processing.NewMemoryRepository()), storage, clock, ids, 0.8)

	pair := circuit.RevisionPair{
		Extraction: circuit.ExtractionRevision{ID: "ext-001", WorkspaceID: "ws-001", ProjectID: "prj-001", UploadID: "up-001", JobID: "job-001", Version: 1, ObjectKey: "ext-key", CreatedAt: clock.now, UpdatedAt: clock.now},
		Circuit:    circuit.CircuitRevision{ID: "cir-001", WorkspaceID: "ws-001", ProjectID: "prj-001", UploadID: "up-001", JobID: "job-001", Version: 1, ObjectKey: "cir-key", Confidence: 0.4, Warnings: []string{"ambiguous-node-label"}, Status: circuit.StatusNeedsReview, Spec: circuit.CircuitSpec{Confidence: 0.4, Warnings: []string{"ambiguous-node-label"}, Status: circuit.StatusNeedsReview}, CreatedAt: clock.now, UpdatedAt: clock.now},
	}
	if err := repo.Save(context.Background(), pair); err != nil {
		t.Fatalf("Save error = %v", err)
	}

	_, err := circuits.SubmitReviewDecision(context.Background(), circuit.SubmitReviewDecisionInput{
		ProjectID:         "prj-001",
		JobID:             "job-001",
		CircuitRevisionID: "cir-001",
		Decision:          circuit.ReviewDecisionReject,
		Note:              "reject",
		ReviewerID:        "reviewer-001",
		ReviewedAt:        time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, processing.ErrJobNotFound) {
		t.Fatalf("SubmitReviewDecision error = %v, want %v", err, processing.ErrJobNotFound)
	}

	storedPair, ok, err := repo.GetByJobID(context.Background(), "job-001")
	if err != nil {
		t.Fatalf("GetByJobID error = %v", err)
	}
	if !ok {
		t.Fatal("expected stored pair")
	}
	if storedPair.Circuit.Status != circuit.StatusNeedsReview {
		t.Fatalf("stored circuit status = %s, want %s", storedPair.Circuit.Status, circuit.StatusNeedsReview)
	}
	if _, ok, err := repo.GetReviewDecision(context.Background(), "cir-001"); err != nil {
		t.Fatalf("GetReviewDecision error = %v", err)
	} else if ok {
		t.Fatal("expected no persisted review audit after rollback")
	}
}

type reviewEnv struct {
	jobs     *processing.Service
	repo     *circuit.MemoryRepository
	storage  storage.ObjectStorage
	circuits *circuit.Service
	clock    fixedClock
	ids      *sequenceIDs
}

type seededReview struct {
	job       processing.Job
	persisted circuit.PersistOutput
}

func newReviewEnv(t *testing.T) reviewEnv {
	t.Helper()
	clock := fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}
	ids := &sequenceIDs{ids: []string{"job-001", "ext-001", "cir-001", "job-002", "ext-002", "cir-002", "job-003", "ext-003", "cir-003", "decision-001", "cir-004", "decision-002", "decision-003", "decision-004"}}
	jobRepo := processing.NewMemoryRepository()
	jobs := processing.NewService(jobRepo, clock, ids)
	repo := circuit.NewMemoryRepository().WithProcessingRepository(jobRepo)
	storage := server.NewMemoryObjectStorage()
	circuits := circuit.NewService(repo, storage, clock, ids, 0.8)
	return reviewEnv{jobs: jobs, repo: repo, storage: storage, circuits: circuits, clock: clock, ids: ids}
}

func (e reviewEnv) seedJob(t *testing.T, projectID string, confidence float64, warnings []string) seededReview {
	t.Helper()
	job, err := e.jobs.CreateJob(context.Background(), "ws-001", projectID, "up-"+projectID+"-"+time.Now().Format("150405.000000000"), 1)
	if err != nil {
		t.Fatalf("CreateJob error = %v", err)
	}
	job, err = e.jobs.Transition(context.Background(), job.ID, processing.JobStatusCreated, processing.JobStatusUploaded)
	if err != nil {
		t.Fatalf("transition to uploaded: %v", err)
	}
	job, err = e.jobs.Transition(context.Background(), job.ID, processing.JobStatusUploaded, processing.JobStatusExtracting)
	if err != nil {
		t.Fatalf("transition to extracting: %v", err)
	}

	persisted, err := e.circuits.Persist(context.Background(), circuit.PersistInput{
		WorkspaceID: "ws-001",
		ProjectID:   projectID,
		UploadID:    job.UploadID,
		Job:         job,
		Result:      circuit.ExtractionResult{Provider: "mock", Confidence: confidence, Warnings: warnings},
	})
	if err != nil {
		t.Fatalf("Persist error = %v", err)
	}
	job, err = e.jobs.MarkCompleted(context.Background(), job.ID, persisted.Status)
	if err != nil {
		t.Fatalf("MarkCompleted error = %v", err)
	}
	return seededReview{job: job, persisted: persisted}
}

func readStoredCircuitSpec(t *testing.T, store interface {
	Get(context.Context, string) (io.ReadCloser, error)
}, key string) circuit.CircuitSpec {
	t.Helper()
	body, err := store.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("get stored circuit spec: %v", err)
	}
	defer body.Close()

	var spec circuit.CircuitSpec
	if err := json.NewDecoder(body).Decode(&spec); err != nil {
		t.Fatalf("decode stored circuit spec: %v", err)
	}
	return spec
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
