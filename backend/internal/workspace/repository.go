package workspace

import (
	"context"
	"errors"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrProjectNotFound   = errors.New("project not found")
)

type Repository interface {
	CreateWorkspace(ctx context.Context, workspace Workspace) error
	CreateProject(ctx context.Context, project Project) error
	GetWorkspace(ctx context.Context, workspaceID string) (Workspace, error)
	GetProject(ctx context.Context, projectID string) (Project, error)
}
