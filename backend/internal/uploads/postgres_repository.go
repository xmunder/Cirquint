package uploads

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateUpload(ctx context.Context, upload Upload) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO uploads (id, workspace_id, project_id, filename, content_type, size_bytes, storage_key, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, upload.ID, upload.WorkspaceID, upload.ProjectID, upload.Filename, upload.ContentType, upload.SizeBytes, upload.StorageKey, upload.Status, upload.CreatedAt, upload.UpdatedAt)
	return err
}

func (r *PostgresRepository) UpdateUploadStatus(ctx context.Context, uploadID, status string, updatedAt time.Time) (Upload, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE uploads
		SET status = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, workspace_id, project_id, filename, content_type, size_bytes, storage_key, status, created_at, updated_at
	`, status, updatedAt, uploadID)
	upload, err := scanUpload(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Upload{}, ErrUploadNotFound
	}
	return upload, err
}

func (r *PostgresRepository) GetUpload(ctx context.Context, uploadID string) (Upload, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, project_id, filename, content_type, size_bytes, storage_key, status, created_at, updated_at
		FROM uploads
		WHERE id = $1
	`, uploadID)
	upload, err := scanUpload(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Upload{}, ErrUploadNotFound
	}
	return upload, err
}

func (r *PostgresRepository) DeleteUpload(ctx context.Context, uploadID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM uploads WHERE id = $1`, uploadID)
	return err
}

type uploadScanner interface {
	Scan(dest ...any) error
}

func scanUpload(scanner uploadScanner) (Upload, error) {
	var upload Upload
	err := scanner.Scan(&upload.ID, &upload.WorkspaceID, &upload.ProjectID, &upload.Filename, &upload.ContentType, &upload.SizeBytes, &upload.StorageKey, &upload.Status, &upload.CreatedAt, &upload.UpdatedAt)
	return upload, err
}
