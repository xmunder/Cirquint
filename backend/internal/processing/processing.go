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
	ErrQueueEmpty              = errors.New("queue empty")
)

type Queue interface {
	Enqueue(ctx context.Context, payload Payload) error
	Dequeue(ctx context.Context) (Payload, error)
}

type Repository interface {
	CreateJob(ctx context.Context, job Job) error
	UpdateJobStatus(ctx context.Context, jobID string, from, to JobStatus, updatedAt time.Time) (Job, error)
	ListJobsByProjectStatus(ctx context.Context, projectID string, status JobStatus) ([]Job, error)
	GetJob(ctx context.Context, jobID string) (Job, error)
	DeleteJob(ctx context.Context, jobID string) error
}

type Service struct {
	repo  Repository
	queue Queue
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

func (s *Service) WithQueue(queue Queue) *Service {
	s.queue = queue
	return s
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

func (s *Service) Enqueue(ctx context.Context, job Job) error {
	if s.queue == nil {
		return nil
	}
	return s.queue.Enqueue(ctx, job.Payload)
}

func (s *Service) Dequeue(ctx context.Context) (Payload, error) {
	if s.queue == nil {
		return Payload{}, ErrQueueEmpty
	}
	return s.queue.Dequeue(ctx)
}

func (s *Service) ClaimQueued(ctx context.Context, payload Payload) (Job, error) {
	return s.Transition(ctx, payload.JobID, JobStatusQueued, JobStatusExtracting)
}

func (s *Service) MarkCompleted(ctx context.Context, jobID string, status JobStatus) (Job, error) {
	return s.Transition(ctx, jobID, JobStatusExtracting, status)
}

func (s *Service) MarkFailed(ctx context.Context, jobID string) (Job, error) {
	return s.Transition(ctx, jobID, JobStatusExtracting, JobStatusFailed)
}

func (s *Service) ResolveReview(ctx context.Context, jobID string, status JobStatus) (Job, error) {
	if status != JobStatusReady && status != JobStatusFailed {
		return Job{}, ErrInvalidStatusTransition
	}
	return s.Transition(ctx, jobID, JobStatusNeedsReview, status)
}

func (s *Service) ListByProjectStatus(ctx context.Context, projectID string, status JobStatus) ([]Job, error) {
	return s.repo.ListJobsByProjectStatus(ctx, projectID, status)
}

func (s *Service) GetJob(ctx context.Context, jobID string) (Job, error) {
	return s.repo.GetJob(ctx, jobID)
}

func (s *Service) DeleteJob(ctx context.Context, jobID string) error {
	return s.repo.DeleteJob(ctx, jobID)
}
