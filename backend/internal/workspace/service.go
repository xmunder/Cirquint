package workspace

import (
	"context"
	"errors"
	"strings"

	"github.com/msi/circuit-storys/backend/internal/platform"
)

var (
	ErrWorkspaceNameRequired = errors.New("workspace name is required")
	ErrProjectNameRequired   = errors.New("project name is required")
)

type Service struct {
	repo  Repository
	clock platform.Clock
	idGen platform.IDGenerator
}

func NewService(repo Repository, clock platform.Clock, idGen platform.IDGenerator) *Service {
	return &Service{repo: repo, clock: clock, idGen: idGen}
}

func (s *Service) CreateWorkspace(ctx context.Context, name string) (Workspace, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Workspace{}, ErrWorkspaceNameRequired
	}

	entity := Workspace{
		ID:        s.idGen.NewID(),
		Name:      name,
		CreatedAt: s.clock.Now(),
	}

	if err := s.repo.CreateWorkspace(ctx, entity); err != nil {
		return Workspace{}, err
	}

	return entity, nil
}

func (s *Service) CreateProject(ctx context.Context, workspaceID, name string) (Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Project{}, ErrProjectNameRequired
	}

	if _, err := s.repo.GetWorkspace(ctx, workspaceID); err != nil {
		return Project{}, err
	}

	entity := Project{
		ID:          s.idGen.NewID(),
		WorkspaceID: workspaceID,
		Name:        name,
		CreatedAt:   s.clock.Now(),
	}

	if err := s.repo.CreateProject(ctx, entity); err != nil {
		return Project{}, err
	}

	return entity, nil
}

func (s *Service) GetProject(ctx context.Context, projectID string) (Project, error) {
	return s.repo.GetProject(ctx, projectID)
}
