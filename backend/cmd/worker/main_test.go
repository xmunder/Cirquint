package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/config"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/provider"
	"github.com/msi/circuit-storys/backend/internal/storage"
	"github.com/msi/circuit-storys/backend/internal/testpostgres"
	"github.com/msi/circuit-storys/backend/internal/uploads"
)

type stubRunner struct {
	err    error
	called bool
}

func (s *stubRunner) ProcessUntilEmpty(context.Context) error {
	s.called = true
	return s.err
}

func TestRunTreatsEmptyQueueAsSuccess(t *testing.T) {
	runner := &stubRunner{err: processing.ErrQueueEmpty}
	if err := run(context.Background(), runner); err != nil {
		t.Fatalf("run error = %v, want nil", err)
	}
	if !runner.called {
		t.Fatal("expected runner to be called")
	}
}

func TestRunReturnsProcessingError(t *testing.T) {
	wantErr := errors.New("boom")
	runner := &stubRunner{err: wantErr}
	if err := run(context.Background(), runner); !errors.Is(err, wantErr) {
		t.Fatalf("run error = %v, want %v", err, wantErr)
	}
}

func TestExecuteUsesConfiguredRunner(t *testing.T) {
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("REDIS_QUEUE_KEY", "jobs")
	t.Setenv("REVIEW_MIN_CONFIDENCE", "0.9")
	t.Setenv("DATABASE_URL", "postgres://app:secret@db.internal:5432/cirquint")
	t.Setenv("OBJECT_STORAGE_BUCKET", "cirquint-dev")
	t.Setenv("OBJECT_STORAGE_ENDPOINT", "https://r2.example.com")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY", "access-key")
	t.Setenv("OBJECT_STORAGE_SECRET_KEY", "secret-key")

	originalFactory := runnerFactory
	originalExec := runnerExec
	t.Cleanup(func() {
		runnerFactory = originalFactory
		runnerExec = originalExec
	})

	stub := &stubRunner{}
	runnerFactory = func(cfg config.Config) (runner, error) {
		if cfg.RedisAddr != "redis:6379" || cfg.RedisQueueKey != "jobs" || cfg.ReviewMinConfidence != 0.9 {
			t.Fatalf("unexpected config %+v", cfg)
		}
		if cfg.DatabaseURL != "postgres://app:secret@db.internal:5432/cirquint" {
			t.Fatalf("DatabaseURL = %q, want loaded env", cfg.DatabaseURL)
		}
		if cfg.ObjectStorage.Bucket != "cirquint-dev" {
			t.Fatalf("ObjectStorage.Bucket = %q, want loaded env", cfg.ObjectStorage.Bucket)
		}
		return stub, nil
	}
	runnerExec = func(ctx context.Context, got runner) error {
		if got != stub {
			t.Fatal("expected execute to use factory runner")
		}
		return nil
	}

	if err := execute(); err != nil {
		t.Fatalf("execute error = %v", err)
	}
}

func TestMainReturnsWhenExecuteSucceeds(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://app:secret@db.internal:5432/cirquint")
	t.Setenv("OBJECT_STORAGE_BUCKET", "cirquint-dev")
	t.Setenv("OBJECT_STORAGE_ENDPOINT", "https://r2.example.com")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY", "access-key")
	t.Setenv("OBJECT_STORAGE_SECRET_KEY", "secret-key")
	originalFactory := runnerFactory
	originalExec := runnerExec
	t.Cleanup(func() {
		runnerFactory = originalFactory
		runnerExec = originalExec
	})

	runnerFactory = func(config.Config) (runner, error) { return &stubRunner{}, nil }
	called := false
	runnerExec = func(context.Context, runner) error {
		called = true
		return nil
	}

	main()
	if !called {
		t.Fatal("expected main to execute runner")
	}
}

func TestBuildRunnerUsesConfiguredRedisAddress(t *testing.T) {
	stubWorkerOpenDatabase(t)
	runner, err := buildRunner(config.Config{
		RedisAddr:           "127.0.0.1:0",
		RedisQueueKey:       "jobs",
		ReviewMinConfidence: 0.9,
		DatabaseURL:         "postgres://app:secret@db.internal:5432/cirquint",
		ObjectStorage: config.ObjectStorageConfig{
			Bucket:    "cirquint-dev",
			Endpoint:  "https://r2.example.com",
			AccessKey: "access-key",
			SecretKey: "secret-key",
		},
	})
	if err != nil {
		t.Fatalf("buildRunner error = %v", err)
	}
	err = run(context.Background(), runner)
	if err == nil {
		t.Fatal("expected dequeue error from configured redis address")
	}
	if !strings.Contains(err.Error(), "127.0.0.1:0") {
		t.Fatalf("error = %q, want redis address in error", err.Error())
	}
}

func TestBuildRunnerRejectsMissingSharedRuntimeConfig(t *testing.T) {
	_, err := buildRunner(config.Config{RedisAddr: "redis:6379", RedisQueueKey: "jobs", ReviewMinConfidence: 0.9})
	if !errors.Is(err, config.ErrDatabaseURLRequired) {
		t.Fatalf("buildRunner error = %v, want %v", err, config.ErrDatabaseURLRequired)
	}
}

func TestBuildRuntimeRunnerProcessesQueuedJobWithPostgresRepositories(t *testing.T) {
	databaseURL := testpostgres.URL(t, "cmd/worker")
	db := openWorkerTestPostgres(t)
	now := time.Date(2026, 6, 1, 18, 0, 0, 0, time.UTC)
	storageRoot := t.TempDir()
	seedWorkerRuntimeJob(t, db, seededWorkerRuntimeJob{
		workspaceID: "ws-001",
		projectID:   "prj-001",
		uploadID:    "up-001",
		jobID:       "job-001",
		createdAt:   now,
	})

	uploadRepo := uploads.NewMemoryRepository()
	seedingStorage := storage.NewFilesystemObjectStorage(storageRoot)
	upload := uploads.Upload{
		ID:          "up-001",
		WorkspaceID: "ws-001",
		ProjectID:   "prj-001",
		Filename:    "diagram.png",
		ContentType: "image/png",
		SizeBytes:   int64(len("png-data")),
		StorageKey:  "workspaces/ws-001/uploads/up-001/v1/source.png",
		Status:      uploads.UploadStatusUploaded,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := uploadRepo.CreateUpload(context.Background(), upload); err != nil {
		t.Fatalf("CreateUpload error = %v", err)
	}
	if _, err := seedingStorage.Put(context.Background(), upload.StorageKey, strings.NewReader("png-data"), storage.ObjectMeta{ContentType: upload.ContentType, SizeBytes: upload.SizeBytes}); err != nil {
		t.Fatalf("seed object storage: %v", err)
	}

	queue := &stubQueue{payloads: []processing.Payload{{JobID: "job-001", ProjectID: "prj-001", UploadID: "up-001", UploadVersion: 1}}}
	runner, err := buildRuntimeRunner(config.Config{
		DatabaseURL:         databaseURL,
		ReviewMinConfidence: 0.8,
		ObjectStorage: config.ObjectStorageConfig{
			Bucket:    "cirquint-dev",
			Endpoint:  "https://r2.example.com",
			AccessKey: "access-key",
			Path:      storageRoot,
			SecretKey: "secret-key",
		},
	}, uploadRepo, nil, provider.StaticProvider{Result: circuit.ExtractionResult{Provider: "mock", Confidence: 0.4, Warnings: []string{"ambiguous-node-label"}}}, queue)
	if err != nil {
		t.Fatalf("buildRuntimeRunner error = %v", err)
	}

	if err := runner.ProcessUntilEmpty(context.Background()); err != nil {
		t.Fatalf("ProcessUntilEmpty error = %v", err)
	}

	jobRepo := processing.NewPostgresRepository(db)
	job, err := jobRepo.GetJob(context.Background(), "job-001")
	if err != nil {
		t.Fatalf("GetJob error = %v", err)
	}
	if job.Status != processing.JobStatusNeedsReview {
		t.Fatalf("job status = %s, want %s", job.Status, processing.JobStatusNeedsReview)
	}

	repo := circuit.NewPostgresRepository(db)
	pair, ok, err := repo.GetByJobID(context.Background(), "job-001")
	if err != nil {
		t.Fatalf("GetByJobID error = %v", err)
	}
	if !ok {
		t.Fatal("expected persisted Postgres revision pair")
	}
	if pair.Circuit.Status != circuit.StatusNeedsReview {
		t.Fatalf("circuit status = %s, want %s", pair.Circuit.Status, circuit.StatusNeedsReview)
	}
}

type stubQueue struct {
	payloads []processing.Payload
	idx      int
}

func (q stubQueue) Enqueue(context.Context, processing.Payload) error {
	return nil
}

func (q *stubQueue) Dequeue(context.Context) (processing.Payload, error) {
	if q.idx >= len(q.payloads) {
		return processing.Payload{}, processing.ErrQueueEmpty
	}
	payload := q.payloads[q.idx]
	q.idx++
	return payload, nil
}

type seededWorkerRuntimeJob struct {
	workspaceID string
	projectID   string
	uploadID    string
	jobID       string
	createdAt   time.Time
}

func openWorkerTestPostgres(t *testing.T) *sql.DB {
	return testpostgres.Open(t, "cmd/worker")
}

func seedWorkerRuntimeJob(t *testing.T, db *sql.DB, job seededWorkerRuntimeJob) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO workspaces (id, name, created_at) VALUES ($1, $2, $3)`, job.workspaceID, "Workspace "+job.workspaceID, job.createdAt); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO projects (id, workspace_id, name, created_at) VALUES ($1, $2, $3, $4)`, job.projectID, job.workspaceID, "Project "+job.projectID, job.createdAt); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO uploads (id, workspace_id, project_id, filename, content_type, size_bytes, storage_key, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`, job.uploadID, job.workspaceID, job.projectID, "diagram.png", "image/png", 8, "workspaces/"+job.workspaceID+"/uploads/"+job.uploadID+"/v1/source.png", uploads.UploadStatusUploaded, job.createdAt, job.createdAt); err != nil {
		t.Fatalf("insert upload: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO processing_jobs (id, workspace_id, project_id, upload_id, status, payload, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)`, job.jobID, job.workspaceID, job.projectID, job.uploadID, processing.JobStatusQueued, `{"job_id":"`+job.jobID+`","project_id":"`+job.projectID+`","upload_id":"`+job.uploadID+`","upload_version":1}`, job.createdAt, job.createdAt); err != nil {
		t.Fatalf("insert processing job: %v", err)
	}
}

func stubWorkerOpenDatabase(t *testing.T) {
	t.Helper()
	original := openDatabase
	t.Cleanup(func() { openDatabase = original })
	openDatabase = func(string) (*sql.DB, error) { return nil, nil }
}
