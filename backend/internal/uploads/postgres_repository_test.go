package uploads

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/testpostgres"
)

func TestPostgresRepositoryRoundTripsUploadLifecycle(t *testing.T) {
	db := openUploadsTestPostgres(t)
	repo := NewPostgresRepository(db)
	now := time.Date(2026, 6, 1, 19, 15, 0, 0, time.UTC)
	seedUploadWorkspace(t, db, now)

	upload := Upload{
		ID:          "up-001",
		WorkspaceID: "ws-001",
		ProjectID:   "prj-001",
		Filename:    "diagram.png",
		ContentType: "image/png",
		SizeBytes:   4,
		StorageKey:  "workspaces/ws-001/uploads/up-001/v1/source.png",
		Status:      UploadStatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.CreateUpload(context.Background(), upload); err != nil {
		t.Fatalf("CreateUpload error = %v", err)
	}

	updatedAt := now.Add(time.Minute)
	updated, err := repo.UpdateUploadStatus(context.Background(), upload.ID, UploadStatusUploaded, updatedAt)
	if err != nil {
		t.Fatalf("UpdateUploadStatus error = %v", err)
	}
	if updated.Status != UploadStatusUploaded || !updated.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("updated upload = %+v", updated)
	}

	got, err := repo.GetUpload(context.Background(), upload.ID)
	if err != nil {
		t.Fatalf("GetUpload error = %v", err)
	}
	if got.StorageKey != upload.StorageKey || got.Status != UploadStatusUploaded {
		t.Fatalf("upload = %+v, want storage/status from lifecycle", got)
	}

	if err := repo.DeleteUpload(context.Background(), upload.ID); err != nil {
		t.Fatalf("DeleteUpload error = %v", err)
	}
	if _, err := repo.GetUpload(context.Background(), upload.ID); err != ErrUploadNotFound {
		t.Fatalf("GetUpload after delete error = %v, want %v", err, ErrUploadNotFound)
	}
}

func openUploadsTestPostgres(t *testing.T) *sql.DB {
	return testpostgres.Open(t, "internal/uploads")
}

func seedUploadWorkspace(t *testing.T, db *sql.DB, now time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO workspaces (id, name, created_at) VALUES ($1, $2, $3)`, "ws-001", "Workspace Alpha", now); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO projects (id, workspace_id, name, created_at) VALUES ($1, $2, $3, $4)`, "prj-001", "ws-001", "Project One", now); err != nil {
		t.Fatalf("insert project: %v", err)
	}
}
