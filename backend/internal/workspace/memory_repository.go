package workspace

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu         sync.RWMutex
	workspaces map[string]Workspace
	projects   map[string]Project
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		workspaces: map[string]Workspace{},
		projects:   map[string]Project{},
	}
}

func (r *MemoryRepository) CreateWorkspace(_ context.Context, workspace Workspace) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workspaces[workspace.ID] = workspace
	return nil
}

func (r *MemoryRepository) CreateProject(_ context.Context, project Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.projects[project.ID] = project
	return nil
}

func (r *MemoryRepository) GetWorkspace(_ context.Context, workspaceID string) (Workspace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	workspace, ok := r.workspaces[workspaceID]
	if !ok {
		return Workspace{}, ErrWorkspaceNotFound
	}
	return workspace, nil
}

func (r *MemoryRepository) GetProject(_ context.Context, projectID string) (Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	project, ok := r.projects[projectID]
	if !ok {
		return Project{}, ErrProjectNotFound
	}
	return project, nil
}
