package circuit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/msi/circuit-storys/backend/internal/processing"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetByJobID(ctx context.Context, jobID string) (RevisionPair, bool, error) {
	return r.lookupLatestPair(ctx, `WHERE cr.job_id = $1`, jobID)
}

func (r *PostgresRepository) GetLatestReviewable(ctx context.Context, projectID, jobID string) (RevisionPair, bool, error) {
	return r.lookupLatestPair(ctx, `WHERE cr.project_id = $1 AND cr.job_id = $2 AND cr.status = $3`, projectID, jobID, StatusNeedsReview)
}

func (r *PostgresRepository) ListLatestReviewable(ctx context.Context, projectID string) ([]RevisionPair, error) {
	rows, err := r.db.QueryContext(ctx, latestPairSelect+`
		WHERE cr.project_id = $1 AND cr.status = $2
		ORDER BY cr.created_at ASC, cr.id ASC
	`, projectID, StatusNeedsReview)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pairs := make([]RevisionPair, 0)
	for rows.Next() {
		pair, scanErr := scanRevisionPair(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		pairs = append(pairs, pair)
	}
	return pairs, rows.Err()
}

func (r *PostgresRepository) Save(ctx context.Context, pair RevisionPair) error {
	extractionWarnings, err := json.Marshal(pair.Extraction.Warnings)
	if err != nil {
		return err
	}
	circuitWarnings, err := json.Marshal(pair.Circuit.Warnings)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO extraction_revisions (id, workspace_id, project_id, upload_id, job_id, version, object_key, confidence, warnings, status, provider_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, $12, $13)
	`, pair.Extraction.ID, pair.Extraction.WorkspaceID, pair.Extraction.ProjectID, pair.Extraction.UploadID, pair.Extraction.JobID, pair.Extraction.Version, pair.Extraction.ObjectKey, pair.Extraction.Confidence, extractionWarnings, statusFromCircuit(pair.Circuit.Status), pair.Extraction.Provider, pair.Extraction.CreatedAt, pair.Extraction.UpdatedAt)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO circuit_revisions (id, workspace_id, project_id, upload_id, job_id, extraction_revision_id, version, object_key, confidence, warnings, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13)
	`, pair.Circuit.ID, pair.Circuit.WorkspaceID, pair.Circuit.ProjectID, pair.Circuit.UploadID, pair.Circuit.JobID, pair.Extraction.ID, pair.Circuit.Version, pair.Circuit.ObjectKey, pair.Circuit.Confidence, circuitWarnings, pair.Circuit.Status, pair.Circuit.CreatedAt, pair.Circuit.UpdatedAt)
	return err
}

func (r *PostgresRepository) SaveReviewDecision(ctx context.Context, decision ReviewDecisionRecord, resolved CircuitRevision) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := ensureReviewableRevision(ctx, tx, decision.ProjectID, decision.JobID, decision.ReviewedCircuitRevisionID); err != nil {
		return err
	}

	if resolved.ID == decision.ReviewedCircuitRevisionID {
		if _, err := tx.ExecContext(ctx, `
			UPDATE circuit_revisions
			SET status = $1, updated_at = $2
			WHERE id = $3
		`, resolved.Status, resolved.UpdatedAt, resolved.ID); err != nil {
			return err
		}
	} else {
		warnings, marshalErr := json.Marshal(resolved.Warnings)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO circuit_revisions (id, workspace_id, project_id, upload_id, job_id, extraction_revision_id, version, object_key, confidence, warnings, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13)
		`, resolved.ID, resolved.WorkspaceID, resolved.ProjectID, resolved.UploadID, resolved.JobID, decision.ExtractionRevisionID, resolved.Version, resolved.ObjectKey, resolved.Confidence, warnings, resolved.Status, resolved.CreatedAt, resolved.UpdatedAt); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE processing_jobs
		SET status = $1, updated_at = $2
		WHERE id = $3 AND status = $4
	`, statusFromCircuit(resolved.Status), resolved.UpdatedAt, decision.JobID, processing.JobStatusNeedsReview); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO circuit_review_decisions (id, workspace_id, project_id, job_id, extraction_revision_id, reviewed_circuit_revision_id, resolved_circuit_revision_id, reviewer_id, decision, note, reviewed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, decision.ID, decision.WorkspaceID, decision.ProjectID, decision.JobID, decision.ExtractionRevisionID, decision.ReviewedCircuitRevisionID, decision.ResolvedCircuitRevisionID, decision.ReviewerID, decision.Decision, decision.Note, decision.ReviewedAt, decision.CreatedAt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetReviewDecision(ctx context.Context, reviewedCircuitRevisionID string) (ReviewDecisionRecord, bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, project_id, job_id, extraction_revision_id, reviewed_circuit_revision_id, resolved_circuit_revision_id, reviewer_id, decision, note, reviewed_at, created_at
		FROM circuit_review_decisions
		WHERE reviewed_circuit_revision_id = $1
	`, reviewedCircuitRevisionID)
	decision, err := scanReviewDecision(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewDecisionRecord{}, false, nil
	}
	return decision, err == nil, err
}

const latestPairSelect = `
	SELECT
		er.id, er.workspace_id, er.project_id, er.upload_id, er.job_id, er.version, er.object_key, er.provider_name, er.confidence, er.warnings, er.created_at, er.updated_at,
		cr.id, cr.workspace_id, cr.project_id, cr.upload_id, cr.job_id, cr.version, cr.object_key, cr.confidence, cr.warnings, cr.status, cr.created_at, cr.updated_at
	FROM circuit_revisions cr
	JOIN extraction_revisions er ON er.id = cr.extraction_revision_id
	JOIN (
		SELECT job_id, MAX(version) AS max_version
		FROM circuit_revisions
		GROUP BY job_id
	) latest ON latest.job_id = cr.job_id AND latest.max_version = cr.version
`

func (r *PostgresRepository) lookupLatestPair(ctx context.Context, where string, args ...any) (RevisionPair, bool, error) {
	query := fmt.Sprintf("%s %s", latestPairSelect, where)
	row := r.db.QueryRowContext(ctx, query, args...)
	pair, err := scanRevisionPair(row)
	if errors.Is(err, sql.ErrNoRows) {
		return RevisionPair{}, false, nil
	}
	return pair, err == nil, err
}

func ensureReviewableRevision(ctx context.Context, tx *sql.Tx, projectID, jobID, revisionID string) error {
	row := tx.QueryRowContext(ctx, `
		SELECT id
		FROM circuit_revisions
		WHERE project_id = $1 AND job_id = $2 AND id = $3 AND status = $4
	`, projectID, jobID, revisionID, StatusNeedsReview)
	var id string
	if err := row.Scan(&id); errors.Is(err, sql.ErrNoRows) {
		return ErrReviewNotFound
	} else {
		return err
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRevisionPair(scanner rowScanner) (RevisionPair, error) {
	var (
		pair            RevisionPair
		extractionWarns []byte
		circuitWarns    []byte
	)
	if err := scanner.Scan(
		&pair.Extraction.ID, &pair.Extraction.WorkspaceID, &pair.Extraction.ProjectID, &pair.Extraction.UploadID, &pair.Extraction.JobID, &pair.Extraction.Version, &pair.Extraction.ObjectKey, &pair.Extraction.Provider, &pair.Extraction.Confidence, &extractionWarns, &pair.Extraction.CreatedAt, &pair.Extraction.UpdatedAt,
		&pair.Circuit.ID, &pair.Circuit.WorkspaceID, &pair.Circuit.ProjectID, &pair.Circuit.UploadID, &pair.Circuit.JobID, &pair.Circuit.Version, &pair.Circuit.ObjectKey, &pair.Circuit.Confidence, &circuitWarns, &pair.Circuit.Status, &pair.Circuit.CreatedAt, &pair.Circuit.UpdatedAt,
	); err != nil {
		return RevisionPair{}, err
	}
	if err := json.Unmarshal(extractionWarns, &pair.Extraction.Warnings); err != nil {
		return RevisionPair{}, err
	}
	if err := json.Unmarshal(circuitWarns, &pair.Circuit.Warnings); err != nil {
		return RevisionPair{}, err
	}
	pair.Extraction.Result = ExtractionResult{Provider: pair.Extraction.Provider, Confidence: pair.Extraction.Confidence, Warnings: append([]string(nil), pair.Extraction.Warnings...)}
	pair.Circuit.Spec = CircuitSpec{Confidence: pair.Circuit.Confidence, Warnings: append([]string(nil), pair.Circuit.Warnings...), Status: pair.Circuit.Status}
	return pair, nil
}

func scanReviewDecision(scanner rowScanner) (ReviewDecisionRecord, error) {
	var decision ReviewDecisionRecord
	err := scanner.Scan(&decision.ID, &decision.WorkspaceID, &decision.ProjectID, &decision.JobID, &decision.ExtractionRevisionID, &decision.ReviewedCircuitRevisionID, &decision.ResolvedCircuitRevisionID, &decision.ReviewerID, &decision.Decision, &decision.Note, &decision.ReviewedAt, &decision.CreatedAt)
	return decision, err
}
