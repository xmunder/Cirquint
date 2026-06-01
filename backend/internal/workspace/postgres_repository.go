package workspace

import (
	"context"
	"database/sql"
	"errors"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateWorkspace(ctx context.Context, workspace Workspace) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, created_at)
		VALUES ($1, $2, $3)
	`, workspace.ID, workspace.Name, workspace.CreatedAt)
	return err
}

func (r *PostgresRepository) CreateProject(ctx context.Context, project Project) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO projects (id, workspace_id, name, created_at)
		VALUES ($1, $2, $3, $4)
	`, project.ID, project.WorkspaceID, project.Name, project.CreatedAt)
	return err
}

func (r *PostgresRepository) GetWorkspace(ctx context.Context, workspaceID string) (Workspace, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, created_at
		FROM workspaces
		WHERE id = $1
	`, workspaceID)
	workspace, err := scanWorkspace(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Workspace{}, ErrWorkspaceNotFound
	}
	return workspace, err
}

func (r *PostgresRepository) GetProject(ctx context.Context, projectID string) (Project, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, created_at
		FROM projects
		WHERE id = $1
	`, projectID)
	project, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrProjectNotFound
	}
	return project, err
}

type workspaceScanner interface {
	Scan(dest ...any) error
}

func scanWorkspace(scanner workspaceScanner) (Workspace, error) {
	var workspace Workspace
	err := scanner.Scan(&workspace.ID, &workspace.Name, &workspace.CreatedAt)
	return workspace, err
}

func scanProject(scanner workspaceScanner) (Project, error) {
	var project Project
	err := scanner.Scan(&project.ID, &project.WorkspaceID, &project.Name, &project.CreatedAt)
	return project, err
}
