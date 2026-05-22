package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/msi/circuit-storys/backend/internal/identity"
	"github.com/msi/circuit-storys/backend/internal/platform"
	"github.com/msi/circuit-storys/backend/internal/platform/httpjson"
	"github.com/msi/circuit-storys/backend/internal/uploads"
	"github.com/msi/circuit-storys/backend/internal/workspace"
)

type Dependencies struct {
	WorkspaceRepo workspace.Repository
	WorkspaceSvc  *workspace.Service
	UploadSvc     *uploads.Service
	Identity      identity.Middleware
	Clock         platform.Clock
	IDGenerator   platform.IDGenerator
}

func New(deps Dependencies) http.Handler {
	repo := deps.WorkspaceRepo
	if repo == nil {
		repo = workspace.NewMemoryRepository()
	}

	clock := deps.Clock
	if clock == nil {
		clock = platform.SystemClock{}
	}

	idGenerator := deps.IDGenerator
	if idGenerator == nil {
		idGenerator = platform.RandomIDGenerator{}
	}

	workspaceSvc := deps.WorkspaceSvc
	if workspaceSvc == nil {
		workspaceSvc = workspace.NewService(repo, clock, idGenerator)
	}

	uploadSvc := deps.UploadSvc
	if uploadSvc == nil {
		uploadSvc = uploads.NewService(workspaceSvc)
	}

	server := &Server{
		workspace: workspaceSvc,
		uploads:   uploadSvc,
	}

	identityMiddleware := deps.Identity
	return identityMiddleware.Wrap(http.HandlerFunc(server.serveHTTP))
}

type Server struct {
	workspace *workspace.Service
	uploads   *uploads.Service
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/workspaces":
		s.handleCreateWorkspace(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/workspaces/") && strings.HasSuffix(r.URL.Path, "/projects"):
		s.handleCreateProject(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/projects/"):
		s.handleProjectRoutes(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/projects/") && strings.HasSuffix(r.URL.Path, "/uploads"):
		s.handleUploadRoute(w, r)
	default:
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "not found"})
	}
}

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpjson.Write(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	entity, err := s.workspace.CreateWorkspace(r.Context(), request.Name)
	if err != nil {
		s.writeWorkspaceError(w, err)
		return
	}

	httpjson.Write(w, http.StatusCreated, entity)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "workspaces" || parts[2] != "projects" {
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	var request struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpjson.Write(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	entity, err := s.workspace.CreateProject(r.Context(), parts[1], request.Name)
	if err != nil {
		s.writeWorkspaceError(w, err)
		return
	}

	httpjson.Write(w, http.StatusCreated, entity)
}

func (s *Server) handleProjectRoutes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "projects" {
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	entity, err := s.workspace.GetProject(r.Context(), parts[1])
	if err != nil {
		s.writeWorkspaceError(w, err)
		return
	}

	httpjson.Write(w, http.StatusOK, entity)
}

func (s *Server) handleUploadRoute(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "projects" || parts[2] != "uploads" {
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	err := s.uploads.PrepareUpload(r.Context(), parts[1])
	switch {
	case err == nil:
		httpjson.Write(w, http.StatusAccepted, map[string]any{"status": "accepted"})
	case errors.Is(err, uploads.ErrProjectIDRequired):
		httpjson.Write(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	case errors.Is(err, workspace.ErrProjectNotFound):
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "project not found"})
	case errors.Is(err, uploads.ErrNotImplemented):
		httpjson.Write(w, http.StatusNotImplemented, map[string]any{
			"error":  "upload flow not implemented",
			"status": "project_verified",
		})
	default:
		httpjson.Write(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
	}
}

func (s *Server) writeWorkspaceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workspace.ErrWorkspaceNameRequired), errors.Is(err, workspace.ErrProjectNameRequired):
		httpjson.Write(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	case errors.Is(err, workspace.ErrWorkspaceNotFound):
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "workspace not found"})
	case errors.Is(err, workspace.ErrProjectNotFound):
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "project not found"})
	default:
		httpjson.Write(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
	}
}
