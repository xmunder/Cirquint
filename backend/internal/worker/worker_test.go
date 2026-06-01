package worker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/provider"
	"github.com/msi/circuit-storys/backend/internal/storage"
	"github.com/msi/circuit-storys/backend/internal/uploads"
	"github.com/msi/circuit-storys/backend/internal/worker"
	"github.com/msi/circuit-storys/backend/internal/workspace"
)

func TestWorkerProcessesReadyNeedsReviewAndFailed(t *testing.T) {
	tests := []struct {
		name          string
		provider      provider.CircuitExtractionProvider
		wantStatus    processing.JobStatus
		wantErr       bool
		wantRevisions bool
	}{
		{
			name:          "ready",
			provider:      provider.StaticProvider{Result: circuit.ExtractionResult{Provider: "mock", Confidence: 0.93, Warnings: []string{"minor"}}},
			wantStatus:    processing.JobStatusReady,
			wantRevisions: true,
		},
		{
			name:          "needs review",
			provider:      provider.StaticProvider{Result: circuit.ExtractionResult{Provider: "mock", Confidence: 0.7, Warnings: []string{"minor"}}},
			wantStatus:    processing.JobStatusNeedsReview,
			wantRevisions: true,
		},
		{
			name:       "failed",
			provider:   provider.StaticProvider{Err: errors.New("provider failed")},
			wantStatus: processing.JobStatusFailed,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newWorkerEnv()
			runner := worker.NewRunner(env.jobs, env.uploadRepo, env.storage, tt.provider, env.circuits)

			jobID, err := env.createQueuedJob(context.Background(), "diagram.png")
			if err != nil {
				t.Fatalf("create queued job: %v", err)
			}

			err = runner.ProcessNext(context.Background())
			if tt.wantErr && err == nil {
				t.Fatal("expected worker error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected worker error: %v", err)
			}

			job, err := env.jobRepo.GetJob(context.Background(), jobID)
			if err != nil {
				t.Fatalf("get job: %v", err)
			}
			if job.Status != tt.wantStatus {
				t.Fatalf("expected status %s, got %s", tt.wantStatus, job.Status)
			}

			_, ok, err := env.circuitRepo.GetByJobID(context.Background(), jobID)
			if err != nil {
				t.Fatalf("get revisions: %v", err)
			}
			if ok != tt.wantRevisions {
				t.Fatalf("expected revisions=%v, got %v", tt.wantRevisions, ok)
			}
		})
	}
}

func TestWorkerPersistsExtractionArtifactsAndKeepsPlanningOutputsOutOfScope(t *testing.T) {
	tests := []struct {
		name       string
		result     circuit.ExtractionResult
		wantStatus processing.JobStatus
	}{
		{
			name:       "ready",
			result:     circuit.ExtractionResult{Provider: "mock", Confidence: 0.93, Warnings: []string{"minor"}},
			wantStatus: processing.JobStatusReady,
		},
		{
			name:       "needs review",
			result:     circuit.ExtractionResult{Provider: "mock", Confidence: 0.7, Warnings: []string{"minor"}},
			wantStatus: processing.JobStatusNeedsReview,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newWorkerEnv()
			runner := worker.NewRunner(env.jobs, env.uploadRepo, env.storage, provider.StaticProvider{Result: tt.result}, env.circuits)

			jobID, err := env.createQueuedJob(context.Background(), "diagram.png")
			if err != nil {
				t.Fatalf("create queued job: %v", err)
			}

			if err := runner.ProcessNext(context.Background()); err != nil {
				t.Fatalf("process next: %v", err)
			}

			pair, ok, err := env.circuitRepo.GetByJobID(context.Background(), jobID)
			if err != nil {
				t.Fatalf("get revisions: %v", err)
			}
			if !ok {
				t.Fatal("expected persisted revisions")
			}

			storedExtraction := readStoredExtractionRevision(t, env.storage, pair.Extraction.ObjectKey)
			if !reflect.DeepEqual(storedExtraction.Result, tt.result) {
				t.Fatalf("expected stored extraction result %+v, got %+v", tt.result, storedExtraction.Result)
			}
			if storedExtraction.Provider != tt.result.Provider || storedExtraction.Confidence != tt.result.Confidence {
				t.Fatalf("expected extraction metadata to mirror result, got %+v", storedExtraction)
			}

			storedSpec := readStoredCircuitSpec(t, env.storage, pair.Circuit.ObjectKey)
			if processing.JobStatus(storedSpec.Status) != tt.wantStatus {
				t.Fatalf("expected stored circuit status %s, got %s", tt.wantStatus, storedSpec.Status)
			}
			if pair.Circuit.AssemblyPlanObjectKey != "" || pair.Circuit.SceneSpecObjectKey != "" || pair.Circuit.ViewerPayloadObjectKey != "" {
				t.Fatal("expected assembly plan, scene spec, and viewer payloads to remain out of scope")
			}

			keys := env.storage.Keys()
			if len(keys) != 3 {
				t.Fatalf("expected only source, extraction, and circuit artifacts, got %d keys: %v", len(keys), keys)
			}
		})
	}
}

func TestWorkerIsIdempotentAcrossDuplicateQueueAttempts(t *testing.T) {
	env := newWorkerEnv()
	runner := worker.NewRunner(
		env.jobs,
		env.uploadRepo,
		env.storage,
		provider.StaticProvider{Result: circuit.ExtractionResult{Provider: "mock", Confidence: 0.95}},
		env.circuits,
	)

	jobID, err := env.createQueuedJob(context.Background(), "diagram.png")
	if err != nil {
		t.Fatalf("create queued job: %v", err)
	}

	if err := runner.ProcessNext(context.Background()); err != nil {
		t.Fatalf("first run: %v", err)
	}

	if _, err := env.jobRepo.UpdateJobStatus(context.Background(), jobID, processing.JobStatusReady, processing.JobStatusQueued, time.Date(2026, 5, 22, 12, 5, 0, 0, time.UTC)); err != nil {
		t.Fatalf("requeue job: %v", err)
	}
	readyJob, err := env.jobRepo.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get ready job: %v", err)
	}
	if err := env.jobs.Enqueue(context.Background(), readyJob); err != nil {
		t.Fatalf("enqueue duplicate payload: %v", err)
	}

	if err := runner.ProcessNext(context.Background()); err != nil {
		t.Fatalf("second run: %v", err)
	}

	pair, ok, err := env.circuitRepo.GetByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get revisions: %v", err)
	}
	if !ok {
		t.Fatal("expected persisted revisions")
	}
	if pair.Extraction.ID != "ext-001" || pair.Circuit.ID != "cir-001" {
		t.Fatalf("expected stable revision ids, got %s and %s", pair.Extraction.ID, pair.Circuit.ID)
	}

	job, err := env.jobRepo.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get final job: %v", err)
	}
	if job.Status != processing.JobStatusReady {
		t.Fatalf("expected ready after retry, got %s", job.Status)
	}
	if pair.Circuit.AssemblyPlanObjectKey != "" || pair.Circuit.SceneSpecObjectKey != "" || pair.Circuit.ViewerPayloadObjectKey != "" {
		t.Fatal("expected out-of-scope artifacts to stay empty")
	}
}

func TestWorkerMarksJobFailedWhenUploadLookupFails(t *testing.T) {
	env := newWorkerEnv()
	runner := worker.NewRunner(env.jobs, env.uploadRepo, env.storage, provider.StaticProvider{Result: circuit.ExtractionResult{Provider: "mock", Confidence: 0.95}}, env.circuits)

	jobID, err := env.createQueuedJob(context.Background(), "diagram.png")
	if err != nil {
		t.Fatalf("create queued job: %v", err)
	}
	job, err := env.jobRepo.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if err := env.uploadRepo.DeleteUpload(context.Background(), job.UploadID); err != nil {
		t.Fatalf("delete upload: %v", err)
	}

	err = runner.ProcessNext(context.Background())
	if err == nil {
		t.Fatal("expected process error")
	}

	job, err = env.jobRepo.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get updated job: %v", err)
	}
	if job.Status != processing.JobStatusFailed {
		t.Fatalf("job status = %s, want %s", job.Status, processing.JobStatusFailed)
	}
}

func TestWorkerProcessUntilEmptyReturnsNilWhenQueueDrains(t *testing.T) {
	env := newWorkerEnv()
	runner := worker.NewRunner(env.jobs, env.uploadRepo, env.storage, provider.StaticProvider{Result: circuit.ExtractionResult{Provider: "mock", Confidence: 0.95}}, env.circuits)

	jobID, err := env.createQueuedJob(context.Background(), "diagram.png")
	if err != nil {
		t.Fatalf("create queued job: %v", err)
	}
	if err := runner.ProcessUntilEmpty(context.Background()); err != nil {
		t.Fatalf("ProcessUntilEmpty error = %v, want nil", err)
	}
	job, err := env.jobRepo.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if job.Status != processing.JobStatusReady {
		t.Fatalf("job status = %s, want %s", job.Status, processing.JobStatusReady)
	}
}

func TestWorkerReturnsReadErrorAndMarksFailed(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}
	ids := &sequenceIDs{ids: []string{"up-001", "job-001", "ext-001", "cir-001"}}
	jobRepo := processing.NewMemoryRepository()
	queue := processing.NewMemoryQueue()
	jobs := processing.NewService(jobRepo, clock, ids).WithQueue(queue)
	uploadRepo := uploads.NewMemoryRepository()
	brokenStorage := &brokenReadStorage{trackingObjectStorage: trackingObjectStorage{objects: map[string][]byte{}}}
	circuitRepo := circuit.NewMemoryRepository()
	circuits := circuit.NewService(circuitRepo, brokenStorage, clock, ids, 0.8)
	uploadSvc := uploads.NewService(fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}}, uploadRepo, brokenStorage, jobs, clock, ids)

	result, err := uploadSvc.Upload(context.Background(), "prj-001", uploads.UploadRequest{
		Filename:    "diagram.png",
		ContentType: "image/png",
		SizeBytes:   int64(len("data")),
		Body:        bytes.NewBufferString("data"),
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	runner := worker.NewRunner(jobs, uploadRepo, brokenStorage, provider.StaticProvider{Result: circuit.ExtractionResult{Provider: "mock", Confidence: 0.95}}, circuits)
	err = runner.ProcessNext(context.Background())
	if err == nil {
		t.Fatal("expected read error")
	}

	job, err := jobRepo.GetJob(context.Background(), result.Job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if job.Status != processing.JobStatusFailed {
		t.Fatalf("job status = %s, want %s", job.Status, processing.JobStatusFailed)
	}
}

type workerEnv struct {
	jobs        *processing.Service
	jobRepo     *processing.MemoryRepository
	uploadRepo  *uploads.MemoryRepository
	storage     *trackingObjectStorage
	circuits    *circuit.Service
	circuitRepo *circuit.MemoryRepository
	uploadSvc   *uploads.Service
}

func newWorkerEnv() workerEnv {
	clock := fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}
	ids := &sequenceIDs{ids: []string{"up-001", "job-001", "ext-001", "cir-001", "ext-002", "cir-002"}}
	jobRepo := processing.NewMemoryRepository()
	queue := processing.NewMemoryQueue()
	jobs := processing.NewService(jobRepo, clock, ids).WithQueue(queue)
	uploadRepo := uploads.NewMemoryRepository()
	objectStorage := newTrackingObjectStorage()
	circuitRepo := circuit.NewMemoryRepository()
	circuits := circuit.NewService(circuitRepo, objectStorage, clock, ids, 0.8)
	uploadSvc := uploads.NewService(fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}}, uploadRepo, objectStorage, jobs, clock, ids)

	return workerEnv{jobs: jobs, jobRepo: jobRepo, uploadRepo: uploadRepo, storage: objectStorage, circuits: circuits, circuitRepo: circuitRepo, uploadSvc: uploadSvc}
}

func (e workerEnv) createQueuedJob(ctx context.Context, filename string) (string, error) {
	result, err := e.uploadSvc.Upload(ctx, "prj-001", uploads.UploadRequest{
		Filename:    filename,
		ContentType: "image/png",
		SizeBytes:   int64(len("data")),
		Body:        bytes.NewBufferString("data"),
	})
	if err != nil {
		return "", err
	}
	return result.Job.ID, nil
}

type fakeProjects struct{ project workspace.Project }

func (f fakeProjects) GetProject(_ context.Context, projectID string) (workspace.Project, error) {
	if f.project.ID != projectID {
		return workspace.Project{}, workspace.ErrProjectNotFound
	}
	return f.project, nil
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

type trackingObjectStorage struct {
	objects map[string][]byte
}

type brokenReadStorage struct{ trackingObjectStorage }

func newTrackingObjectStorage() *trackingObjectStorage {
	return &trackingObjectStorage{objects: map[string][]byte{}}
}

func (s *brokenReadStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	if _, ok := s.objects[key]; !ok {
		return nil, storage.ErrObjectNotFound
	}
	return io.NopCloser(errReader{}), nil
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func (s *trackingObjectStorage) Put(_ context.Context, key string, body io.Reader, _ storage.ObjectMeta) (storage.StoredObject, error) {
	payload, err := io.ReadAll(body)
	if err != nil {
		return storage.StoredObject{}, err
	}
	s.objects[key] = payload
	return storage.StoredObject{Key: key, ETag: "mem"}, nil
}

func (s *trackingObjectStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	payload, ok := s.objects[key]
	if !ok {
		return nil, storage.ErrObjectNotFound
	}
	return io.NopCloser(bytes.NewReader(payload)), nil
}

func (s *trackingObjectStorage) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func (s *trackingObjectStorage) Keys() []string {
	keys := make([]string, 0, len(s.objects))
	for key := range s.objects {
		keys = append(keys, key)
	}
	return keys
}

func readStoredExtractionRevision(t *testing.T, store storage.ObjectStorage, key string) circuit.ExtractionRevision {
	t.Helper()
	body, err := store.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("get stored extraction revision: %v", err)
	}
	defer body.Close()

	var revision circuit.ExtractionRevision
	if err := json.NewDecoder(body).Decode(&revision); err != nil {
		t.Fatalf("decode stored extraction revision: %v", err)
	}
	return revision
}

func readStoredCircuitSpec(t *testing.T, store storage.ObjectStorage, key string) circuit.CircuitSpec {
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
