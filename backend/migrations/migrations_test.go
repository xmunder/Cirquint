package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCircuitReviewDecisionMigrationExists(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join(".", "0002_circuit_review_decisions.sql"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	contents := string(payload)
	if !strings.Contains(contents, "CREATE TABLE circuit_review_decisions") {
		t.Fatalf("migration missing review decision table: %s", contents)
	}
	if !strings.Contains(contents, "UNIQUE (reviewed_circuit_revision_id)") {
		t.Fatalf("migration missing reviewed revision uniqueness: %s", contents)
	}
	if !strings.Contains(contents, "resolved_circuit_revision_id") {
		t.Fatalf("migration missing resolved revision linkage: %s", contents)
	}
	if !strings.Contains(contents, "reviewer_id") {
		t.Fatalf("migration missing reviewer audit column: %s", contents)
	}
}

func TestCircuitReviewDecisionMigrationAddsQueryIndexes(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join(".", "0002_circuit_review_decisions.sql"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	contents := string(payload)
	if !strings.Contains(contents, "circuit_review_decisions_project_created_at_idx") {
		t.Fatalf("migration missing project/created_at index: %s", contents)
	}
	if !strings.Contains(contents, "ON circuit_review_decisions(project_id, created_at)") {
		t.Fatalf("migration missing project/created_at index columns: %s", contents)
	}
	if !strings.Contains(contents, "circuit_review_decisions_job_id_idx") {
		t.Fatalf("migration missing job index: %s", contents)
	}
	if !strings.Contains(contents, "ON circuit_review_decisions(job_id)") {
		t.Fatalf("migration missing job index columns: %s", contents)
	}
}
