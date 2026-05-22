package processing

import "time"

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
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	UploadID  string    `json:"upload_id"`
	Status    JobStatus `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Repository interface{}
