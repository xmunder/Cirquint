package uploads

import (
	"context"
	"errors"
	"strings"

	"github.com/msi/circuit-storys/backend/internal/workspace"
)

var (
	ErrProjectIDRequired = errors.New("project_id is required")
	ErrNotImplemented    = errors.New("upload flow not implemented")
)

type ProjectLookup interface {
	GetProject(ctx context.Context, projectID string) (workspace.Project, error)
}

type Service struct {
	projects ProjectLookup
}

func NewService(projects ProjectLookup) *Service {
	return &Service{projects: projects}
}

func (s *Service) PrepareUpload(ctx context.Context, projectID string) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return ErrProjectIDRequired
	}

	if _, err := s.projects.GetProject(ctx, projectID); err != nil {
		return err
	}

	return ErrNotImplemented
}
