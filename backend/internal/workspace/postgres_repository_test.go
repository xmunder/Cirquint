package workspace

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/testpostgres"
)

func TestPostgresRepositoryCreatesAndLoadsWorkspaceAndProject(t *testing.T) {
	db := openWorkspaceTestPostgres(t)
	repo := NewPostgresRepository(db)
	now := time.Date(2026, 6, 1, 19, 0, 0, 0, time.UTC)

	workspace := Workspace{ID: "ws-001", Name: "Workspace Alpha", CreatedAt: now}
	if err := repo.CreateWorkspace(context.Background(), workspace); err != nil {
		t.Fatalf("CreateWorkspace error = %v", err)
	}

	project := Project{ID: "prj-001", WorkspaceID: workspace.ID, Name: "Project One", CreatedAt: now}
	if err := repo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject error = %v", err)
	}

	gotWorkspace, err := repo.GetWorkspace(context.Background(), workspace.ID)
	if err != nil {
		t.Fatalf("GetWorkspace error = %v", err)
	}
	if gotWorkspace.ID != workspace.ID || gotWorkspace.Name != workspace.Name {
		t.Fatalf("workspace = %+v, want %+v", gotWorkspace, workspace)
	}

	gotProject, err := repo.GetProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("GetProject error = %v", err)
	}
	if gotProject.ID != project.ID || gotProject.WorkspaceID != project.WorkspaceID || gotProject.Name != project.Name {
		t.Fatalf("project = %+v, want %+v", gotProject, project)
	}

	if _, err := repo.GetProject(context.Background(), "missing-project"); err != ErrProjectNotFound {
		t.Fatalf("GetProject missing error = %v, want %v", err, ErrProjectNotFound)
	}
}

func openWorkspaceTestPostgres(t *testing.T) *sql.DB {
	return testpostgres.Open(t, "internal/workspace")
}
