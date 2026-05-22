package uploads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/storage"
	"github.com/msi/circuit-storys/backend/internal/workspace"
)

var (
	ErrProjectIDRequired = errors.New("project_id is required")
	ErrFileRequired      = errors.New("file is required")
	ErrUnsupportedImage  = errors.New("unsupported image content type")
)

const (
	UploadStatusCreated  = "created"
	UploadStatusUploaded = "uploaded"
	UploadVersion        = 1
)

type ProjectLookup interface {
	GetProject(ctx context.Context, projectID string) (workspace.Project, error)
}

type Upload struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	ProjectID   string    `json:"project_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StorageKey  string    `json:"storage_key"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Repository interface {
	CreateUpload(ctx context.Context, upload Upload) error
	UpdateUploadStatus(ctx context.Context, uploadID, status string, updatedAt time.Time) (Upload, error)
	GetUpload(ctx context.Context, uploadID string) (Upload, error)
	DeleteUpload(ctx context.Context, uploadID string) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

type UploadRequest struct {
	Filename    string
	ContentType string
	SizeBytes   int64
	Body        io.Reader
}

type UploadResult struct {
	Upload Upload         `json:"upload"`
	Job    processing.Job `json:"job"`
}

type Service struct {
	projects ProjectLookup
	repo     Repository
	storage  storage.ObjectStorage
	jobs     *processing.Service
	clock    Clock
	idGen    IDGenerator
}

func NewService(projects ProjectLookup, repo Repository, objectStorage storage.ObjectStorage, jobs *processing.Service, clock Clock, idGen IDGenerator) *Service {
	return &Service{
		projects: projects,
		repo:     repo,
		storage:  objectStorage,
		jobs:     jobs,
		clock:    clock,
		idGen:    idGen,
	}
}

func (s *Service) Upload(ctx context.Context, projectID string, request UploadRequest) (UploadResult, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return UploadResult{}, ErrProjectIDRequired
	}
	if strings.TrimSpace(request.Filename) == "" || request.Body == nil {
		return UploadResult{}, ErrFileRequired
	}
	if !isSupportedContentType(request.ContentType, request.Filename) {
		return UploadResult{}, ErrUnsupportedImage
	}

	project, err := s.projects.GetProject(ctx, projectID)
	if err != nil {
		return UploadResult{}, err
	}

	now := s.clock.Now()
	upload := Upload{
		ID:          s.idGen.NewID(),
		WorkspaceID: project.WorkspaceID,
		ProjectID:   project.ID,
		Filename:    request.Filename,
		ContentType: normalizeContentType(request.ContentType, request.Filename),
		SizeBytes:   request.SizeBytes,
		Status:      UploadStatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	upload.StorageKey = storage.UploadObjectKey(upload.WorkspaceID, upload.ID, upload.Filename)

	if err := s.repo.CreateUpload(ctx, upload); err != nil {
		return UploadResult{}, err
	}

	job, err := s.jobs.CreateJob(ctx, upload.WorkspaceID, upload.ProjectID, upload.ID, UploadVersion)
	if err != nil {
		_ = s.repo.DeleteUpload(ctx, upload.ID)
		return UploadResult{}, err
	}

	if _, err := s.storage.Put(ctx, upload.StorageKey, request.Body, storage.ObjectMeta{
		ContentType: upload.ContentType,
		SizeBytes:   upload.SizeBytes,
	}); err != nil {
		_ = s.jobs.DeleteJob(ctx, job.ID)
		_ = s.repo.DeleteUpload(ctx, upload.ID)
		return UploadResult{}, err
	}

	upload, err = s.repo.UpdateUploadStatus(ctx, upload.ID, UploadStatusUploaded, s.clock.Now())
	if err != nil {
		_ = s.jobs.DeleteJob(ctx, job.ID)
		return UploadResult{}, err
	}

	job, err = s.jobs.Transition(ctx, job.ID, processing.JobStatusCreated, processing.JobStatusUploaded)
	if err != nil {
		_ = s.jobs.DeleteJob(ctx, job.ID)
		return UploadResult{}, err
	}

	job, err = s.jobs.Transition(ctx, job.ID, processing.JobStatusUploaded, processing.JobStatusQueued)
	if err != nil {
		_ = s.jobs.DeleteJob(ctx, job.ID)
		return UploadResult{}, err
	}

	return UploadResult{Upload: upload, Job: job}, nil
}

func ReadFormFile(file multipart.File, header *multipart.FileHeader) (UploadRequest, error) {
	if file == nil || header == nil {
		return UploadRequest{}, ErrFileRequired
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mimeTypeFromFilename(header.Filename)
	}

	return UploadRequest{
		Filename:    header.Filename,
		ContentType: contentType,
		SizeBytes:   header.Size,
		Body:        file,
	}, nil
}

func isSupportedContentType(contentType, filename string) bool {
	switch normalizeContentType(contentType, filename) {
	case "image/png", "image/jpeg", "image/webp":
		return true
	default:
		return false
	}
}

func normalizeContentType(contentType, filename string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType != "" {
		if idx := strings.Index(contentType, ";"); idx >= 0 {
			contentType = strings.TrimSpace(contentType[:idx])
		}
	}
	if contentType != "" {
		return contentType
	}
	return mimeTypeFromFilename(filename)
}

func mimeTypeFromFilename(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return fmt.Sprintf("application/%s", strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), "."))
	}
}
