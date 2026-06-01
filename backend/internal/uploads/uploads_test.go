package uploads_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/storage"
	"github.com/msi/circuit-storys/backend/internal/uploads"
	"github.com/msi/circuit-storys/backend/internal/workspace"
)

type multipartBuffer struct{ *bytes.Reader }

func (multipartBuffer) Close() error { return nil }

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
	putErr      error
	lastKey     string
	deletedKeys []string
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

func (f *fakeObjectStorage) Delete(_ context.Context, key string) error {
	f.deletedKeys = append(f.deletedKeys, key)
	return nil
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
	deleted   []string
	createErr error
	updateErr error
}

func (r *failingUploadRepo) CreateUpload(_ context.Context, upload uploads.Upload) error {
	if r.createErr != nil {
		return r.createErr
	}
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

func (r *failingUploadRepo) DeleteUpload(_ context.Context, uploadID string) error {
	r.deleted = append(r.deleted, uploadID)
	if r.stored.ID == uploadID {
		r.stored = uploads.Upload{}
	}
	return nil
}

func TestUploadDBFailureAfterStorageDeletesStoredObject(t *testing.T) {
	jobRepo := processing.NewMemoryRepository()
	objectStorage := &fakeObjectStorage{}
	uploadRepo := &failingUploadRepo{updateErr: errors.New("db failed")}
	service := uploads.NewService(
		fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}},
		uploadRepo,
		objectStorage,
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

	if len(objectStorage.deletedKeys) != 1 {
		t.Fatalf("expected one delete call, got %d", len(objectStorage.deletedKeys))
	}
	if objectStorage.deletedKeys[0] != "workspaces/ws-001/uploads/up-001/v1/source.png" {
		t.Fatalf("unexpected deleted key %q", objectStorage.deletedKeys[0])
	}

	if _, err := jobRepo.GetJob(context.Background(), "job-001"); !errors.Is(err, processing.ErrJobNotFound) {
		t.Fatalf("expected job cleanup, got %v", err)
	}
	if len(uploadRepo.deleted) != 1 || uploadRepo.deleted[0] != "up-001" {
		t.Fatalf("expected upload cleanup call for up-001, got %v", uploadRepo.deleted)
	}
	if _, err := uploadRepo.GetUpload(context.Background(), "up-001"); !errors.Is(err, uploads.ErrUploadNotFound) {
		t.Fatalf("expected upload cleanup in repository, got %v", err)
	}
}

func TestUploadRejectsInvalidInput(t *testing.T) {
	service := uploads.NewService(
		fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}},
		uploads.NewMemoryRepository(),
		&fakeObjectStorage{},
		processing.NewService(processing.NewMemoryRepository(), fixedClock{now: time.Now()}, &sequenceIDs{}),
		fixedClock{now: time.Now()},
		&sequenceIDs{},
	)

	tests := []struct {
		name      string
		projectID string
		request   uploads.UploadRequest
		wantErr   error
	}{
		{name: "missing project", request: uploads.UploadRequest{Filename: "diagram.png", ContentType: "image/png", Body: bytes.NewBufferString("data")}, wantErr: uploads.ErrProjectIDRequired},
		{name: "missing file", projectID: "prj-001", request: uploads.UploadRequest{}, wantErr: uploads.ErrFileRequired},
		{name: "unsupported type", projectID: "prj-001", request: uploads.UploadRequest{Filename: "diagram.gif", ContentType: "image/gif", Body: bytes.NewBufferString("data")}, wantErr: uploads.ErrUnsupportedImage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Upload(context.Background(), tt.projectID, tt.request)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Upload error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadFormFileUsesHeaderFallbackContentType(t *testing.T) {
	header := &multipart.FileHeader{
		Filename: "diagram.jpeg",
		Size:     4,
		Header:   textproto.MIMEHeader{},
	}
	request, err := uploads.ReadFormFile(multipartBuffer{Reader: bytes.NewReader([]byte("data"))}, header)
	if err != nil {
		t.Fatalf("ReadFormFile error = %v", err)
	}
	if request.ContentType != "image/jpeg" {
		t.Fatalf("ContentType = %q, want image/jpeg", request.ContentType)
	}
}

func TestReadFormFileRejectsMissingInputs(t *testing.T) {
	if _, err := uploads.ReadFormFile(nil, nil); !errors.Is(err, uploads.ErrFileRequired) {
		t.Fatalf("ReadFormFile error = %v, want %v", err, uploads.ErrFileRequired)
	}
}

func TestReadFormFilePreservesExplicitContentType(t *testing.T) {
	header := &multipart.FileHeader{
		Filename: "diagram.png",
		Size:     4,
		Header:   textproto.MIMEHeader{"Content-Type": []string{"image/png; charset=utf-8"}},
	}
	request, err := uploads.ReadFormFile(multipartBuffer{Reader: bytes.NewReader([]byte("data"))}, header)
	if err != nil {
		t.Fatalf("ReadFormFile error = %v", err)
	}
	if request.ContentType != "image/png; charset=utf-8" {
		t.Fatalf("ContentType = %q, want explicit header value", request.ContentType)
	}
}

func TestUploadReturnsProjectLookupError(t *testing.T) {
	service := uploads.NewService(
		fakeProjects{project: workspace.Project{ID: "other", WorkspaceID: "ws-001", Name: "Project One"}},
		uploads.NewMemoryRepository(),
		&fakeObjectStorage{},
		processing.NewService(processing.NewMemoryRepository(), fixedClock{now: time.Now()}, &sequenceIDs{ids: []string{"job-001"}}),
		fixedClock{now: time.Now()},
		&sequenceIDs{ids: []string{"up-001"}},
	)

	_, err := service.Upload(context.Background(), "prj-001", uploads.UploadRequest{
		Filename:    "diagram.png",
		ContentType: "image/png",
		SizeBytes:   4,
		Body:        bytes.NewBufferString("data"),
	})
	if !errors.Is(err, workspace.ErrProjectNotFound) {
		t.Fatalf("Upload error = %v, want %v", err, workspace.ErrProjectNotFound)
	}
}

func TestUploadReturnsRepositoryCreateError(t *testing.T) {
	wantErr := errors.New("create upload failed")
	service := uploads.NewService(
		fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}},
		&failingUploadRepo{createErr: wantErr},
		&fakeObjectStorage{},
		processing.NewService(processing.NewMemoryRepository(), fixedClock{now: time.Now()}, &sequenceIDs{ids: []string{"job-001"}}),
		fixedClock{now: time.Now()},
		&sequenceIDs{ids: []string{"up-001"}},
	)

	_, err := service.Upload(context.Background(), "prj-001", uploads.UploadRequest{
		Filename:    "diagram.png",
		ContentType: "image/png",
		SizeBytes:   4,
		Body:        bytes.NewBufferString("data"),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Upload error = %v, want %v", err, wantErr)
	}
}

func TestUploadCleansUpWhenEnqueueFails(t *testing.T) {
	jobRepo := processing.NewMemoryRepository()
	objectStorage := &fakeObjectStorage{}
	queue := failingQueue{err: errors.New("enqueue failed")}
	jobs := processing.NewService(jobRepo, fixedClock{now: time.Now()}, &sequenceIDs{ids: []string{"job-001"}}).WithQueue(queue)
	uploadRepo := uploads.NewMemoryRepository()
	service := uploads.NewService(
		fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}},
		uploadRepo,
		objectStorage,
		jobs,
		fixedClock{now: time.Now()},
		&sequenceIDs{ids: []string{"up-001"}},
	)

	_, err := service.Upload(context.Background(), "prj-001", uploads.UploadRequest{
		Filename:    "diagram.png",
		ContentType: "image/png",
		SizeBytes:   4,
		Body:        bytes.NewBufferString("data"),
	})
	if err == nil {
		t.Fatal("expected enqueue failure")
	}
	if _, err := jobRepo.GetJob(context.Background(), "job-001"); !errors.Is(err, processing.ErrJobNotFound) {
		t.Fatalf("expected job cleanup, got %v", err)
	}
	if _, err := uploadRepo.GetUpload(context.Background(), "up-001"); !errors.Is(err, uploads.ErrUploadNotFound) {
		t.Fatalf("expected upload cleanup, got %v", err)
	}
	if len(objectStorage.deletedKeys) != 1 {
		t.Fatalf("expected stored object delete, got %d deletes", len(objectStorage.deletedKeys))
	}
}

func TestUploadInfersContentTypeFromFilename(t *testing.T) {
	service := uploads.NewService(
		fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}},
		uploads.NewMemoryRepository(),
		&fakeObjectStorage{},
		processing.NewService(processing.NewMemoryRepository(), fixedClock{now: time.Now()}, &sequenceIDs{ids: []string{"job-001"}}),
		fixedClock{now: time.Now()},
		&sequenceIDs{ids: []string{"up-001"}},
	)

	result, err := service.Upload(context.Background(), "prj-001", uploads.UploadRequest{
		Filename:  "diagram.webp",
		SizeBytes: 4,
		Body:      bytes.NewBufferString("data"),
	})
	if err != nil {
		t.Fatalf("Upload error = %v", err)
	}
	if result.Upload.ContentType != "image/webp" {
		t.Fatalf("ContentType = %q, want image/webp", result.Upload.ContentType)
	}
}

func TestUploadNormalizesContentTypeParameters(t *testing.T) {
	service := uploads.NewService(
		fakeProjects{project: workspace.Project{ID: "prj-001", WorkspaceID: "ws-001", Name: "Project One"}},
		uploads.NewMemoryRepository(),
		&fakeObjectStorage{},
		processing.NewService(processing.NewMemoryRepository(), fixedClock{now: time.Now()}, &sequenceIDs{ids: []string{"job-001"}}),
		fixedClock{now: time.Now()},
		&sequenceIDs{ids: []string{"up-001"}},
	)

	result, err := service.Upload(context.Background(), "prj-001", uploads.UploadRequest{
		Filename:    "diagram.png",
		ContentType: "image/png; charset=utf-8",
		SizeBytes:   4,
		Body:        bytes.NewBufferString("data"),
	})
	if err != nil {
		t.Fatalf("Upload error = %v", err)
	}
	if result.Upload.ContentType != "image/png" {
		t.Fatalf("ContentType = %q, want image/png", result.Upload.ContentType)
	}
}

type failingQueue struct{ err error }

func (q failingQueue) Enqueue(context.Context, processing.Payload) error { return q.err }
func (q failingQueue) Dequeue(context.Context) (processing.Payload, error) {
	return processing.Payload{}, processing.ErrQueueEmpty
}
