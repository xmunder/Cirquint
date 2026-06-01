package circuit_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/testpostgres"
)

func TestPostgresRepositorySaveReviewDecisionRollsBackWhenJobTransitionFails(t *testing.T) {
	db := openTestPostgres(t)
	repo := circuit.NewPostgresRepository(db)
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	seedReviewableJob(t, db, seededPostgresReviewJob{
		workspaceID:   "ws-001",
		projectID:     "prj-001",
		uploadID:      "up-001",
		jobID:         "job-001",
		jobStatus:     processing.JobStatusReady,
		extractionID:  "ext-001",
		circuitID:     "cir-001",
		objectKeyBase: "job-001",
		confidence:    0.4,
		warnings:      []string{"ambiguous-node-label"},
		createdAt:     now,
	})

	err := repo.SaveReviewDecision(context.Background(), circuit.ReviewDecisionRecord{
		ID:                        "decision-001",
		WorkspaceID:               "ws-001",
		ProjectID:                 "prj-001",
		JobID:                     "job-001",
		ExtractionRevisionID:      "ext-001",
		ReviewedCircuitRevisionID: "cir-001",
		ResolvedCircuitRevisionID: "cir-001",
		ReviewerID:                "reviewer-001",
		Decision:                  circuit.ReviewDecisionReject,
		Note:                      "reject after review",
		ReviewedAt:                now.Add(time.Hour),
		CreatedAt:                 now.Add(time.Hour),
	}, circuit.CircuitRevision{
		ID:          "cir-001",
		WorkspaceID: "ws-001",
		ProjectID:   "prj-001",
		UploadID:    "up-001",
		JobID:       "job-001",
		Version:     1,
		ObjectKey:   "circuits/job-001-v1.json",
		Confidence:  0.4,
		Warnings:    []string{"ambiguous-node-label"},
		Status:      circuit.StatusFailed,
		Spec:        circuit.CircuitSpec{Confidence: 0.4, Warnings: []string{"ambiguous-node-label"}, Status: circuit.StatusFailed},
		CreatedAt:   now,
		UpdatedAt:   now.Add(time.Hour),
	})
	if !errors.Is(err, processing.ErrInvalidStatusTransition) {
		t.Fatalf("SaveReviewDecision error = %v, want %v", err, processing.ErrInvalidStatusTransition)
	}

	if _, ok, err := repo.GetReviewDecision(context.Background(), "cir-001"); err != nil {
		t.Fatalf("GetReviewDecision error = %v", err)
	} else if ok {
		t.Fatal("expected no stored review decision after rollback")
	}

	var status string
	if err := db.QueryRowContext(context.Background(), `SELECT status FROM circuit_revisions WHERE id = $1`, "cir-001").Scan(&status); err != nil {
		t.Fatalf("query stored circuit status: %v", err)
	}
	if status != circuit.StatusNeedsReview {
		t.Fatalf("stored circuit status = %s, want %s", status, circuit.StatusNeedsReview)
	}
}

func TestPostgresRepositoryPreservesReviewAuditAcrossLaterRetry(t *testing.T) {
	db := openTestPostgres(t)
	repo := circuit.NewPostgresRepository(db)
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	seedReviewableJob(t, db, seededPostgresReviewJob{
		workspaceID:   "ws-001",
		projectID:     "prj-001",
		uploadID:      "up-001",
		jobID:         "job-001",
		jobStatus:     processing.JobStatusNeedsReview,
		extractionID:  "ext-001",
		circuitID:     "cir-001",
		objectKeyBase: "job-001",
		confidence:    0.4,
		warnings:      []string{"ambiguous-node-label"},
		createdAt:     now,
	})

	err := repo.SaveReviewDecision(context.Background(), circuit.ReviewDecisionRecord{
		ID:                        "decision-001",
		WorkspaceID:               "ws-001",
		ProjectID:                 "prj-001",
		JobID:                     "job-001",
		ExtractionRevisionID:      "ext-001",
		ReviewedCircuitRevisionID: "cir-001",
		ResolvedCircuitRevisionID: "cir-002",
		ReviewerID:                "reviewer-001",
		Decision:                  circuit.ReviewDecisionApprove,
		Note:                      "fixed labels",
		ReviewedAt:                now.Add(time.Hour),
		CreatedAt:                 now.Add(time.Hour),
	}, circuit.CircuitRevision{
		ID:          "cir-002",
		WorkspaceID: "ws-001",
		ProjectID:   "prj-001",
		UploadID:    "up-001",
		JobID:       "job-001",
		Version:     2,
		ObjectKey:   "circuits/job-001-v2.json",
		Confidence:  0.97,
		Warnings:    []string{"human-corrected"},
		Status:      circuit.StatusReady,
		Spec:        circuit.CircuitSpec{Confidence: 0.97, Warnings: []string{"human-corrected"}, Status: circuit.StatusReady},
		CreatedAt:   now.Add(time.Hour),
		UpdatedAt:   now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("SaveReviewDecision error = %v", err)
	}

	seedReviewableJob(t, db, seededPostgresReviewJob{
		workspaceID:   "ws-001",
		projectID:     "prj-001",
		uploadID:      "up-001",
		jobID:         "job-002",
		jobStatus:     processing.JobStatusNeedsReview,
		extractionID:  "ext-002",
		circuitID:     "cir-003",
		objectKeyBase: "job-002",
		confidence:    0.33,
		warnings:      []string{"ambiguous-node-label"},
		createdAt:     now.Add(2 * time.Hour),
	})

	audit, ok, err := repo.GetReviewDecision(context.Background(), "cir-001")
	if err != nil {
		t.Fatalf("GetReviewDecision error = %v", err)
	}
	if !ok {
		t.Fatal("expected stored review decision for original revision")
	}
	if audit.ResolvedCircuitRevisionID != "cir-002" {
		t.Fatalf("audit resolved_circuit_revision_id = %s, want cir-002", audit.ResolvedCircuitRevisionID)
	}

	var (
		storedObjectKey  string
		storedConfidence float64
	)
	if err := db.QueryRowContext(context.Background(), `SELECT object_key, confidence FROM extraction_revisions WHERE id = $1`, "ext-001").Scan(&storedObjectKey, &storedConfidence); err != nil {
		t.Fatalf("query original extraction revision: %v", err)
	}
	if storedObjectKey != "extractions/job-001-v1.json" {
		t.Fatalf("original extraction object_key = %s, want extractions/job-001-v1.json", storedObjectKey)
	}
	if storedConfidence != 0.4 {
		t.Fatalf("original extraction confidence = %v, want 0.4", storedConfidence)
	}

	var extractionCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM extraction_revisions WHERE upload_id = $1`, "up-001").Scan(&extractionCount); err != nil {
		t.Fatalf("count extraction revisions: %v", err)
	}
	if extractionCount != 2 {
		t.Fatalf("extraction revision count = %d, want 2", extractionCount)
	}
	if _, ok, err := repo.GetLatestReviewable(context.Background(), "prj-001", "job-002"); err != nil {
		t.Fatalf("GetLatestReviewable retry error = %v", err)
	} else if !ok {
		t.Fatal("expected later retry to remain separately reviewable")
	}
}

type seededPostgresReviewJob struct {
	workspaceID   string
	projectID     string
	uploadID      string
	jobID         string
	jobStatus     processing.JobStatus
	extractionID  string
	circuitID     string
	objectKeyBase string
	confidence    float64
	warnings      []string
	createdAt     time.Time
}

func openTestPostgres(t *testing.T) *sql.DB {
	return testpostgres.Open(t, "internal/circuit")
}

func seedReviewableJob(t *testing.T, db *sql.DB, job seededPostgresReviewJob) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO workspaces (id, name, created_at) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING`, job.workspaceID, "Workspace "+job.workspaceID, job.createdAt); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO projects (id, workspace_id, name, created_at) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING`, job.projectID, job.workspaceID, "Project "+job.projectID, job.createdAt); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO uploads (id, workspace_id, project_id, filename, content_type, size_bytes, storage_key, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT (id) DO NOTHING`, job.uploadID, job.workspaceID, job.projectID, job.uploadID+".png", "image/png", 128, "uploads/"+job.uploadID, "uploaded", job.createdAt, job.createdAt); err != nil {
		t.Fatalf("insert upload: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO processing_jobs (id, workspace_id, project_id, upload_id, status, payload, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)`, job.jobID, job.workspaceID, job.projectID, job.uploadID, job.jobStatus, `{"job_id":"`+job.jobID+`","project_id":"`+job.projectID+`","upload_id":"`+job.uploadID+`","upload_version":1}`, job.createdAt, job.createdAt); err != nil {
		t.Fatalf("insert processing job: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO extraction_revisions (id, workspace_id, project_id, upload_id, job_id, version, object_key, confidence, warnings, status, provider_name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, $12, $13)`, job.extractionID, job.workspaceID, job.projectID, job.uploadID, job.jobID, 1, "extractions/"+job.objectKeyBase+"-v1.json", job.confidence, warningsJSON(job.warnings), processing.JobStatusNeedsReview, "mock", job.createdAt, job.createdAt); err != nil {
		t.Fatalf("insert extraction revision: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO circuit_revisions (id, workspace_id, project_id, upload_id, job_id, extraction_revision_id, version, object_key, confidence, warnings, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13)`, job.circuitID, job.workspaceID, job.projectID, job.uploadID, job.jobID, job.extractionID, 1, "circuits/"+job.objectKeyBase+"-v1.json", job.confidence, warningsJSON(job.warnings), circuit.StatusNeedsReview, job.createdAt, job.createdAt); err != nil {
		t.Fatalf("insert circuit revision: %v", err)
	}
}

func warningsJSON(warnings []string) string {
	if len(warnings) == 0 {
		return `[]`
	}
	quoted := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		quoted = append(quoted, `"`+warning+`"`)
	}
	return `[` + strings.Join(quoted, ",") + `]`
}
