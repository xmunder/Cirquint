package server

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/msi/circuit-storys/backend/internal/identity"
	"github.com/msi/circuit-storys/backend/internal/platform"
	"github.com/msi/circuit-storys/backend/internal/platform/httpjson"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/storage"
	"github.com/msi/circuit-storys/backend/internal/uploads"
	"github.com/msi/circuit-storys/backend/internal/workspace"
)

type Dependencies struct {
	WorkspaceRepo workspace.Repository
	WorkspaceSvc  *workspace.Service
	UploadRepo    uploads.Repository
	UploadSvc     *uploads.Service
	JobRepo       processing.Repository
	JobSvc        *processing.Service
	Queue         processing.Queue
	ObjectStorage storage.ObjectStorage
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

	jobRepo := deps.JobRepo
	if jobRepo == nil {
		jobRepo = processing.NewMemoryRepository()
	}

	jobSvc := deps.JobSvc
	if jobSvc == nil {
		jobSvc = processing.NewService(jobRepo, clock, idGenerator)
	}
	if deps.Queue != nil {
		jobSvc = jobSvc.WithQueue(deps.Queue)
	}

	uploadRepo := deps.UploadRepo
	if uploadRepo == nil {
		uploadRepo = uploads.NewMemoryRepository()
	}

	objectStorage := deps.ObjectStorage
	if objectStorage == nil {
		objectStorage = NewMemoryObjectStorage()
	}

	uploadSvc := deps.UploadSvc
	if uploadSvc == nil {
		uploadSvc = uploads.NewService(workspaceSvc, uploadRepo, objectStorage, jobSvc, clock, idGenerator)
	}

	server := &Server{
		workspace: workspaceSvc,
		uploads:   uploadSvc,
		jobs:      jobSvc,
	}

	identityMiddleware := deps.Identity
	return identityMiddleware.Wrap(http.HandlerFunc(server.serveHTTP))
}

type Server struct {
	workspace *workspace.Service
	uploads   *uploads.Service
	jobs      *processing.Service
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/workspaces":
		s.handleCreateWorkspace(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/workspaces/") && strings.HasSuffix(r.URL.Path, "/projects"):
		s.handleCreateProject(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/projects/"):
		s.handleProjectRoutes(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/jobs/"):
		s.handleJobRoute(w, r)
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

	file, header, err := r.FormFile("file")
	if err != nil {
		s.writeUploadError(w, err)
		return
	}
	defer file.Close()

	request, err := uploads.ReadFormFile(file, header)
	if err != nil {
		s.writeUploadError(w, err)
		return
	}

	result, err := s.uploads.Upload(r.Context(), parts[1], request)
	switch {
	case err == nil:
		httpjson.Write(w, http.StatusCreated, result)
	case errors.Is(err, uploads.ErrProjectIDRequired):
		httpjson.Write(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	case errors.Is(err, workspace.ErrProjectNotFound):
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "project not found"})
	default:
		s.writeUploadError(w, err)
	}
}

func (s *Server) handleJobRoute(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "jobs" {
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	job, err := s.jobs.GetJob(r.Context(), parts[1])
	if err != nil {
		s.writeJobError(w, err)
		return
	}

	httpjson.Write(w, http.StatusOK, job)
}

func (s *Server) writeUploadError(w http.ResponseWriter, err error) {
	if errors.Is(err, multipart.ErrMessageTooLarge) || errors.Is(err, http.ErrMissingFile) {
		httpjson.Write(w, http.StatusBadRequest, map[string]any{"error": uploads.ErrFileRequired.Error()})
		return
	}

	switch {
	case errors.Is(err, uploads.ErrFileRequired), errors.Is(err, uploads.ErrUnsupportedImage):
		httpjson.Write(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	default:
		httpjson.Write(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
	}
}

func (s *Server) writeJobError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, processing.ErrJobNotFound):
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "job not found"})
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
