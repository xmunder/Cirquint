package processing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateJob(ctx context.Context, job Job) error {
	payload, err := json.Marshal(job.Payload)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO processing_jobs (id, workspace_id, project_id, upload_id, status, payload, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
	`, job.ID, job.WorkspaceID, job.ProjectID, job.UploadID, job.Status, payload, job.CreatedAt, job.UpdatedAt)
	return err
}

func (r *PostgresRepository) UpdateJobStatus(ctx context.Context, jobID string, from, to JobStatus, updatedAt time.Time) (Job, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE processing_jobs
		SET status = $1, updated_at = $2
		WHERE id = $3 AND status = $4
		RETURNING id, workspace_id, project_id, upload_id, status, payload, created_at, updated_at
	`, to, updatedAt, jobID, from)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		if _, lookupErr := r.GetJob(ctx, jobID); lookupErr != nil {
			return Job{}, lookupErr
		}
		return Job{}, ErrInvalidStatusTransition
	}
	return job, err
}

func (r *PostgresRepository) ListJobsByProjectStatus(ctx context.Context, projectID string, status JobStatus) ([]Job, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, workspace_id, project_id, upload_id, status, payload, created_at, updated_at
		FROM processing_jobs
		WHERE project_id = $1 AND status = $2
		ORDER BY updated_at ASC, id ASC
	`, projectID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]Job, 0)
	for rows.Next() {
		job, scanErr := scanJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *PostgresRepository) GetJob(ctx context.Context, jobID string) (Job, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, project_id, upload_id, status, payload, created_at, updated_at
		FROM processing_jobs
		WHERE id = $1
	`, jobID)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrJobNotFound
	}
	return job, err
}

func (r *PostgresRepository) DeleteJob(ctx context.Context, jobID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM processing_jobs WHERE id = $1`, jobID)
	return err
}

type jobScanner interface {
	Scan(dest ...any) error
}

func scanJob(scanner jobScanner) (Job, error) {
	var (
		job     Job
		payload []byte
	)
	if err := scanner.Scan(&job.ID, &job.WorkspaceID, &job.ProjectID, &job.UploadID, &job.Status, &payload, &job.CreatedAt, &job.UpdatedAt); err != nil {
		return Job{}, err
	}
	if err := json.Unmarshal(payload, &job.Payload); err != nil {
		return Job{}, err
	}
	return job, nil
}
