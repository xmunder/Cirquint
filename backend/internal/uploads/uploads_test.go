package uploads_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/storage"
	"github.com/msi/circuit-storys/backend/internal/uploads"
	"github.com/msi/circuit-storys/backend/internal/workspace"
)

func TestUploadSuccessQueuesJob(t *testing.T) {
	uploadRepo := uploads.NewMemoryRepository()
	jobRepo := processing.NewMemoryRepository()
	storage := &fakeObjectStorage{}
	service := uploads.NewService(
		fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}},
		uploadRepo,
		storage,
		processing.NewService(jobRepo, fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}, &sequenceIDs{ids: []string{"job-001"}}),
		fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)},
		&sequenceIDs{ids: []string{"up-001"}},
	)

	result, err := service.Upload(context.Background(), "prj-001", uploads.UploadRequest{
		Filename:    "diagram.png",
		ContentType: "image/png",
		SizeBytes:   4,
		Body:        bytes.NewBufferString("data"),
	})
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	if result.Job.Status != processing.JobStatusQueued {
		t.Fatalf("expected queued status, got %s", result.Job.Status)
	}
	if result.Upload.Status != uploads.UploadStatusUploaded {
		t.Fatalf("expected uploaded status, got %s", result.Upload.Status)
	}
	if storage.lastKey != "workspaces/ws-001/uploads/up-001/v1/source.png" {
		t.Fatalf("unexpected storage key %q", storage.lastKey)
	}

	storedJob, err := jobRepo.GetJob(context.Background(), result.Job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if storedJob.Status != processing.JobStatusQueued {
		t.Fatalf("expected persisted queued status, got %s", storedJob.Status)
	}
}

func TestUploadStorageFailureDoesNotLeaveQueuedJob(t *testing.T) {
	uploadRepo := uploads.NewMemoryRepository()
	jobRepo := processing.NewMemoryRepository()
	service := uploads.NewService(
		fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}},
		uploadRepo,
		&fakeObjectStorage{putErr: errors.New("boom")},
		processing.NewService(jobRepo, fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}, &sequenceIDs{ids: []string{"job-001"}}),
		fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)},
		&sequenceIDs{ids: []string{"up-001"}},
	)

	_, err := service.Upload(context.Background(), "prj-001", uploads.UploadRequest{
		Filename:    "diagram.png",
		ContentType: "image/png",
		SizeBytes:   4,
		Body:        bytes.NewBufferString("data"),
	})
	if err == nil {
		t.Fatal("expected upload failure")
	}

	if _, err := jobRepo.GetJob(context.Background(), "job-001"); !errors.Is(err, processing.ErrJobNotFound) {
		t.Fatalf("expected job cleanup, got %v", err)
	}
	if _, err := uploadRepo.GetUpload(context.Background(), "up-001"); !errors.Is(err, uploads.ErrUploadNotFound) {
		t.Fatalf("expected upload cleanup, got %v", err)
	}
}

func TestUploadDBFailureAfterStorageDoesNotLeaveQueuedJob(t *testing.T) {
	jobRepo := processing.NewMemoryRepository()
	service := uploads.NewService(
		fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}},
		&failingUploadRepo{updateErr: errors.New("db failed")},
		&fakeObjectStorage{},
		processing.NewService(jobRepo, fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}, &sequenceIDs{ids: []string{"job-001"}}),
		fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)},
		&sequenceIDs{ids: []string{"up-001"}},
	)

	_, err := service.Upload(context.Background(), "prj-001", uploads.UploadRequest{
		Filename:    "diagram.png",
		ContentType: "image/png",
		SizeBytes:   4,
		Body:        bytes.NewBufferString("data"),
	})
	if err == nil {
		t.Fatal("expected upload failure")
	}

	if _, err := jobRepo.GetJob(context.Background(), "job-001"); !errors.Is(err, processing.ErrJobNotFound) {
		t.Fatalf("expected job cleanup, got %v", err)
	}
}

type fakeProjects struct{ project workspace.Project }

func (f fakeProjects) GetProject(_ context.Context, projectID string) (workspace.Project, error) {
	if f.project.ID != projectID {
		return workspace.Project{}, workspace.ErrProjectNotFound
	}
	return f.project, nil
}

type fakeObjectStorage struct {
	putErr  error
	lastKey string
}

func (f *fakeObjectStorage) Put(_ context.Context, key string, body io.Reader, _ storage.ObjectMeta) (storage.StoredObject, error) {
	f.lastKey = key
	if f.putErr != nil {
		return storage.StoredObject{}, f.putErr
	}
	_, _ = io.ReadAll(body)
	return storage.StoredObject{Key: key, ETag: "etag-001"}, nil
}

func (f *fakeObjectStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	return nil, nil
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

type failingUploadRepo struct {
	stored    uploads.Upload
	updateErr error
}

func (r *failingUploadRepo) CreateUpload(_ context.Context, upload uploads.Upload) error {
	r.stored = upload
	return nil
}

func (r *failingUploadRepo) UpdateUploadStatus(_ context.Context, _ string, _ string, _ time.Time) (uploads.Upload, error) {
	return uploads.Upload{}, r.updateErr
}

func (r *failingUploadRepo) GetUpload(_ context.Context, uploadID string) (uploads.Upload, error) {
	if r.stored.ID != uploadID {
		return uploads.Upload{}, uploads.ErrUploadNotFound
	}
	return r.stored, nil
}

func (r *failingUploadRepo) DeleteUpload(_ context.Context, _ string) error { return nil }
