package processing

import (
	"context"
	"errors"
	"time"
)

type JobStatus string

const (
	JobStatusCreated     JobStatus = "created"
	JobStatusUploaded    JobStatus = "uploaded"
	JobStatusQueued      JobStatus = "queued"
	JobStatusExtracting  JobStatus = "extracting"
	JobStatusNeedsReview JobStatus = "needs_review"
	JobStatusReady       JobStatus = "ready"
	JobStatusFailed      JobStatus = "failed"
)

type Job struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	ProjectID   string    `json:"project_id"`
	UploadID    string    `json:"upload_id"`
	Status      JobStatus `json:"status"`
	Payload     Payload   `json:"payload"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Payload struct {
	JobID         string `json:"job_id"`
	ProjectID     string `json:"project_id"`
	UploadID      string `json:"upload_id"`
	UploadVersion int    `json:"upload_version"`
}

var (
	ErrJobNotFound             = errors.New("job not found")
	ErrInvalidStatusTransition = errors.New("invalid job status transition")
)

type Repository interface {
	CreateJob(ctx context.Context, job Job) error
	UpdateJobStatus(ctx context.Context, jobID string, from, to JobStatus, updatedAt time.Time) (Job, error)
	GetJob(ctx context.Context, jobID string) (Job, error)
	DeleteJob(ctx context.Context, jobID string) error
}

type Service struct {
	repo  Repository
	clock Clock
	idGen IDGenerator
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

func NewService(repo Repository, clock Clock, idGen IDGenerator) *Service {
	return &Service{repo: repo, clock: clock, idGen: idGen}
}

func (s *Service) CreateJob(ctx context.Context, workspaceID, projectID, uploadID string, uploadVersion int) (Job, error) {
	now := s.clock.Now()
	jobID := s.idGen.NewID()
	job := Job{
		ID:          jobID,
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		UploadID:    uploadID,
		Status:      JobStatusCreated,
		Payload: Payload{
			JobID:         jobID,
			ProjectID:     projectID,
			UploadID:      uploadID,
			UploadVersion: uploadVersion,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateJob(ctx, job); err != nil {
		return Job{}, err
	}

	return job, nil
}

func (s *Service) Transition(ctx context.Context, jobID string, from, to JobStatus) (Job, error) {
	return s.repo.UpdateJobStatus(ctx, jobID, from, to, s.clock.Now())
}

func (s *Service) GetJob(ctx context.Context, jobID string) (Job, error) {
	return s.repo.GetJob(ctx, jobID)
}

func (s *Service) DeleteJob(ctx context.Context, jobID string) error {
	return s.repo.DeleteJob(ctx, jobID)
}
