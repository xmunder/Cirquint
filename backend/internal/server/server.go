package server

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/msi/circuit-storys/backend/internal/circuit"
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
	CircuitRepo   circuit.Repository
	CircuitSvc    *circuit.Service
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

	circuitSvc := deps.CircuitSvc
	if circuitSvc == nil {
		circuitRepo := deps.CircuitRepo
		if circuitRepo == nil {
			circuitRepo = circuit.NewMemoryRepository()
		}
		circuitSvc = circuit.NewService(circuitRepo, objectStorage, clock, idGenerator, 0)
	}

	server := &Server{
		workspace: workspaceSvc,
		uploads:   uploadSvc,
		jobs:      jobSvc,
		circuits:  circuitSvc,
	}

	identityMiddleware := deps.Identity
	return identityMiddleware.Wrap(http.HandlerFunc(server.serveHTTP))
}

type Server struct {
	workspace *workspace.Service
	uploads   *uploads.Service
	jobs      *processing.Service
	circuits  *circuit.Service
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/workspaces":
		s.handleCreateWorkspace(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/workspaces/") && strings.HasSuffix(r.URL.Path, "/projects"):
		s.handleCreateProject(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/projects/") && strings.HasSuffix(r.URL.Path, "/uploads"):
		s.handleUploadRoute(w, r)
	case strings.HasPrefix(r.URL.Path, "/projects/"):
		s.handleProjectRoutes(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/jobs/"):
		s.handleJobRoute(w, r)
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
	if len(parts) < 2 || parts[0] != "projects" {
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	if len(parts) == 3 && parts[2] == "circuit-reviews" && r.Method == http.MethodGet {
		s.handleReviewQueue(w, r, parts[1])
		return
	}

	if len(parts) == 4 && parts[2] == "circuit-reviews" && r.Method == http.MethodGet {
		s.handleReviewDetail(w, r, parts[1], parts[3])
		return
	}

	if len(parts) == 5 && parts[2] == "circuit-reviews" && parts[4] == "decision" && r.Method == http.MethodPost {
		s.handleReviewDecision(w, r, parts[1], parts[3])
		return
	}

	if len(parts) != 2 {
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

func (s *Server) handleReviewQueue(w http.ResponseWriter, r *http.Request, projectID string) {
	items, err := s.circuits.ListReviewQueue(r.Context(), projectID)
	if err != nil {
		s.writeReviewError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleReviewDetail(w http.ResponseWriter, r *http.Request, projectID, jobID string) {
	detail, err := s.circuits.GetReviewDetail(r.Context(), projectID, jobID)
	if err != nil {
		s.writeReviewError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, detail)
}

func (s *Server) handleReviewDecision(w http.ResponseWriter, r *http.Request, projectID, jobID string) {
	var request struct {
		CircuitRevisionID string                 `json:"circuit_revision_id"`
		Decision          circuit.ReviewDecision `json:"decision"`
		Note              string                 `json:"note"`
		ReviewedAt        string                 `json:"reviewed_at"`
		CorrectedSpec     *circuit.CircuitSpec   `json:"corrected_spec"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpjson.Write(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	reviewedAt, err := time.Parse(time.RFC3339, request.ReviewedAt)
	if err != nil {
		httpjson.Write(w, http.StatusBadRequest, map[string]any{"error": "invalid reviewed_at"})
		return
	}

	actor := identity.ActorFromContext(r.Context())
	result, err := s.circuits.SubmitReviewDecision(r.Context(), circuit.SubmitReviewDecisionInput{
		ProjectID:            projectID,
		JobID:                jobID,
		CircuitRevisionID:    request.CircuitRevisionID,
		Decision:             request.Decision,
		Note:                 request.Note,
		ReviewerID:           actor.ID,
		ReviewedAt:           reviewedAt,
		CorrectedCircuitSpec: request.CorrectedSpec,
	})
	if err != nil {
		s.writeReviewError(w, err)
		return
	}

	httpjson.Write(w, http.StatusOK, result)
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

func (s *Server) writeReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, circuit.ErrInvalidReviewDecision),
		errors.Is(err, circuit.ErrInvalidReviewNote),
		errors.Is(err, circuit.ErrInvalidReviewerID),
		errors.Is(err, circuit.ErrCorrectedSpecNotAllowed),
		errors.Is(err, circuit.ErrInvalidCircuitSpec):
		httpjson.Write(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	case errors.Is(err, circuit.ErrReviewNotFound):
		httpjson.Write(w, http.StatusNotFound, map[string]any{"error": "review not found"})
	case errors.Is(err, circuit.ErrStaleReviewRevision), errors.Is(err, circuit.ErrRevisionAlreadyReviewed):
		httpjson.Write(w, http.StatusConflict, map[string]any{"error": err.Error()})
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
