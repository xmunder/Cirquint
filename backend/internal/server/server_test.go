package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/platform"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/server"
	"github.com/msi/circuit-storys/backend/internal/storage"
)

func TestWorkspaceProjectCreationContract(t *testing.T) {
	handler := newTestServer()

	workspaceBody := performJSONRequest(t, handler, http.MethodPost, "/workspaces", map[string]string{
		"name": "Workspace Alpha",
	})

	workspaceID := workspaceBody["id"].(string)
	if workspaceID == "" {
		t.Fatal("expected workspace id")
	}

	projectBody := performJSONRequest(t, handler, http.MethodPost, "/workspaces/"+workspaceID+"/projects", map[string]string{
		"name": "Project One",
	})

	projectID := projectBody["id"].(string)
	if projectID == "" {
		t.Fatal("expected project id")
	}

	response := performRequest(t, handler, http.MethodGet, "/projects/"+projectID, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", response.Code)
	}

	actual := normalizeBody(t, response.Body.Bytes())
	assertGolden(t, "workspace_project_creation.golden", actual)
}

func TestUploadRejectsInvalidProject(t *testing.T) {
	handler := newTestServer()

	response := performMultipartRequest(t, handler, http.MethodPost, "/projects/missing-project/uploads", "diagram.png", "image/png", []byte("data"))
	if response.Code != http.StatusNotFound {
		t.Fatalf("unexpected status %d", response.Code)
	}

	actual := normalizeBody(t, response.Body.Bytes())
	assertGolden(t, "upload_invalid_project.golden", actual)
}

func TestUploadCreatesQueuedJobAndPollingContract(t *testing.T) {
	handler := newTestServerWithIDs("ws-001", "prj-001", "up-001", "job-001")

	workspaceBody := performJSONRequest(t, handler, http.MethodPost, "/workspaces", map[string]string{"name": "Workspace Alpha"})
	workspaceID := workspaceBody["id"].(string)
	projectBody := performJSONRequest(t, handler, http.MethodPost, "/workspaces/"+workspaceID+"/projects", map[string]string{"name": "Project One"})
	projectID := projectBody["id"].(string)

	uploadResponse := performMultipartRequest(t, handler, http.MethodPost, "/projects/"+projectID+"/uploads", "diagram.png", "image/png", []byte("data"))
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("unexpected upload status %d", uploadResponse.Code)
	}

	var uploadBody map[string]any
	if err := json.Unmarshal(uploadResponse.Body.Bytes(), &uploadBody); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	job := uploadBody["job"].(map[string]any)
	if got := job["status"]; got != "queued" {
		t.Fatalf("expected queued status, got %v", got)
	}

	jobID := job["id"].(string)
	statusResponse := performRequest(t, handler, http.MethodGet, "/jobs/"+jobID, nil)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("unexpected job status code %d", statusResponse.Code)
	}

	actual := normalizeBody(t, statusResponse.Body.Bytes())
	assertGolden(t, "job_status_queued.golden", actual)
}

func TestRoutesReturnExpectedErrors(t *testing.T) {
	handler := newTestServer()

	t.Run("workspace invalid json", func(t *testing.T) {
		response := performRequest(t, handler, http.MethodPost, "/workspaces", bytes.NewBufferString("{"))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("job not found", func(t *testing.T) {
		response := performRequest(t, handler, http.MethodGet, "/jobs/missing", nil)
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
	})

	t.Run("upload missing file", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/projects/prj-001/uploads", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("unknown route", func(t *testing.T) {
		response := performRequest(t, handler, http.MethodGet, "/missing", nil)
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
	})

	t.Run("project invalid json", func(t *testing.T) {
		workspaceBody := performJSONRequest(t, handler, http.MethodPost, "/workspaces", map[string]string{"name": "Workspace Alpha"})
		workspaceID := workspaceBody["id"].(string)
		response := performRequest(t, handler, http.MethodPost, "/workspaces/"+workspaceID+"/projects", bytes.NewBufferString("{"))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("upload unsupported type", func(t *testing.T) {
		workspaceBody := performJSONRequest(t, handler, http.MethodPost, "/workspaces", map[string]string{"name": "Workspace Beta"})
		workspaceID := workspaceBody["id"].(string)
		projectBody := performJSONRequest(t, handler, http.MethodPost, "/workspaces/"+workspaceID+"/projects", map[string]string{"name": "Project Two"})
		projectID := projectBody["id"].(string)
		response := performMultipartRequest(t, handler, http.MethodPost, "/projects/"+projectID+"/uploads", "diagram.gif", "image/gif", []byte("data"))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("project not found", func(t *testing.T) {
		response := performRequest(t, handler, http.MethodGet, "/projects/missing-project", nil)
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
	})

	t.Run("workspace name required", func(t *testing.T) {
		response := performJSONRequestExpectingStatus(t, handler, http.MethodPost, "/workspaces", map[string]string{"name": ""}, http.StatusBadRequest)
		if response["error"] == "" {
			t.Fatal("expected validation error message")
		}
	})

	t.Run("workspace not found for project creation", func(t *testing.T) {
		response := performJSONRequestExpectingStatus(t, handler, http.MethodPost, "/workspaces/missing/projects", map[string]string{"name": "Project Ghost"}, http.StatusNotFound)
		if response["error"] != "workspace not found" {
			t.Fatalf("error = %v, want workspace not found", response["error"])
		}
	})
}

func TestCircuitReviewQueueListsProjectItemsOnly(t *testing.T) {
	env := newReviewHTTPEnv()
	queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})
	env.seedJob(t, "prj-001", 0.98, nil)
	env.seedJob(t, "prj-002", 0.3, []string{"ambiguous-node-label"})

	response := performRequest(t, env.handler, http.MethodGet, "/projects/prj-001/circuit-reviews", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(body.Items))
	}
	if got := body.Items[0]["job_id"]; got != queued.job.ID {
		t.Fatalf("job_id = %v, want %s", got, queued.job.ID)
	}
	if got := body.Items[0]["circuit_revision_id"]; got != queued.persisted.Circuit.ID {
		t.Fatalf("circuit_revision_id = %v, want %s", got, queued.persisted.Circuit.ID)
	}
	if got := body.Items[0]["upload_id"]; got != queued.job.UploadID {
		t.Fatalf("upload_id = %v, want %s", got, queued.job.UploadID)
	}
}

func TestCircuitReviewQueueReturnsEmptyWhenProjectHasNoPendingReviewWork(t *testing.T) {
	env := newReviewHTTPEnv()
	env.seedJob(t, "prj-001", 0.98, nil)
	env.seedJob(t, "prj-002", 0.3, []string{"ambiguous-node-label"})

	response := performRequest(t, env.handler, http.MethodGet, "/projects/prj-001/circuit-reviews", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(body.Items))
	}
}

func TestCircuitReviewDetailReturnsReviewArtifactsOnly(t *testing.T) {
	env := newReviewHTTPEnv()
	queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})

	response := performRequest(t, env.handler, http.MethodGet, "/projects/prj-001/circuit-reviews/"+queued.job.ID, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, forbidden := range []string{"assembly_plan", "scene_spec", "viewer_payload"} {
		if _, ok := body[forbidden]; ok {
			t.Fatalf("response unexpectedly included %q", forbidden)
		}
	}
	job, ok := body["job"].(map[string]any)
	if !ok {
		t.Fatal("expected job object")
	}
	if got := job["id"]; got != queued.job.ID {
		t.Fatalf("job.id = %v, want %s", got, queued.job.ID)
	}
	circuitBody, ok := body["circuit"].(map[string]any)
	if !ok {
		t.Fatal("expected circuit object")
	}
	if got := circuitBody["status"]; got != circuit.StatusNeedsReview {
		t.Fatalf("circuit.status = %v, want %s", got, circuit.StatusNeedsReview)
	}
	warnings, ok := body["warnings"].([]any)
	if !ok || len(warnings) != 1 || warnings[0] != "ambiguous-node-label" {
		t.Fatalf("warnings = %+v, want [ambiguous-node-label]", body["warnings"])
	}

	notFound := performRequest(t, env.handler, http.MethodGet, "/projects/prj-002/circuit-reviews/"+queued.job.ID, nil)
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("cross-project status = %d, want %d", notFound.Code, http.StatusNotFound)
	}
}

func TestCircuitReviewDecisionRouteValidatesAndMapsErrors(t *testing.T) {
	t.Run("approve persists reviewer and clears queue", func(t *testing.T) {
		env := newReviewHTTPEnvWithIDs("cir-approve-001", "decision-001")
		queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})

		response := performJSONRequestExpectingStatus(t, env.handler, http.MethodPost, "/projects/prj-001/circuit-reviews/"+queued.job.ID+"/decision", map[string]any{
			"circuit_revision_id": queued.persisted.Circuit.ID,
			"decision":            "approve",
			"note":                "fixed labels",
			"reviewed_at":         "2026-05-23T09:00:00Z",
			"corrected_spec": map[string]any{
				"confidence": 0.99,
				"warnings":   []string{"human-corrected"},
			},
		}, http.StatusOK)

		decision, ok := response["decision"].(map[string]any)
		if !ok {
			t.Fatal("expected decision object")
		}
		if got := decision["reviewer_id"]; got != "anonymous" {
			t.Fatalf("reviewer_id = %v, want anonymous", got)
		}

		queueResponse := performRequest(t, env.handler, http.MethodGet, "/projects/prj-001/circuit-reviews", nil)
		if queueResponse.Code != http.StatusOK {
			t.Fatalf("queue status = %d, want %d", queueResponse.Code, http.StatusOK)
		}
		var queueBody struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(queueResponse.Body.Bytes(), &queueBody); err != nil {
			t.Fatalf("decode queue response: %v", err)
		}
		if len(queueBody.Items) != 0 {
			t.Fatalf("len(items) after approve = %d, want 0", len(queueBody.Items))
		}
	})

	t.Run("rejects invalid request payloads", func(t *testing.T) {
		env := newReviewHTTPEnv()
		queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})

		cases := []struct {
			name   string
			body   any
			status int
		}{
			{
				name: "invalid decision",
				body: map[string]any{
					"circuit_revision_id": queued.persisted.Circuit.ID,
					"decision":            "maybe",
					"note":                "nope",
					"reviewed_at":         "2026-05-23T09:00:00Z",
				},
				status: http.StatusBadRequest,
			},
			{
				name: "reject with corrected spec",
				body: map[string]any{
					"circuit_revision_id": queued.persisted.Circuit.ID,
					"decision":            "reject",
					"note":                "rejecting",
					"reviewed_at":         "2026-05-23T09:00:00Z",
					"corrected_spec": map[string]any{
						"confidence": 0.5,
					},
				},
				status: http.StatusBadRequest,
			},
			{
				name: "invalid reviewed_at",
				body: map[string]any{
					"circuit_revision_id": queued.persisted.Circuit.ID,
					"decision":            "reject",
					"note":                "rejecting",
					"reviewed_at":         "not-a-time",
				},
				status: http.StatusBadRequest,
			},
			{
				name: "invalid corrected spec",
				body: map[string]any{
					"circuit_revision_id": queued.persisted.Circuit.ID,
					"decision":            "approve",
					"note":                "trying",
					"reviewed_at":         "2026-05-23T09:00:00Z",
					"corrected_spec": map[string]any{
						"confidence": 2,
					},
				},
				status: http.StatusBadRequest,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				response := performJSONRequestExpectingStatus(t, env.handler, http.MethodPost, "/projects/prj-001/circuit-reviews/"+queued.job.ID+"/decision", tc.body, tc.status)
				if response["error"] == "" {
					t.Fatal("expected error response")
				}
			})
		}
	})

	t.Run("maps stale and cross-project errors", func(t *testing.T) {
		env := newReviewHTTPEnvWithIDs("decision-001")
		queued := env.seedJob(t, "prj-001", 0.4, []string{"ambiguous-node-label"})

		stale := performJSONRequestExpectingStatus(t, env.handler, http.MethodPost, "/projects/prj-001/circuit-reviews/"+queued.job.ID+"/decision", map[string]any{
			"circuit_revision_id": "cir-stale",
			"decision":            "reject",
			"note":                "stale",
			"reviewed_at":         "2026-05-23T09:00:00Z",
		}, http.StatusConflict)
		if stale["error"] == "" {
			t.Fatal("expected stale error body")
		}

		notFound := performJSONRequestExpectingStatus(t, env.handler, http.MethodPost, "/projects/prj-002/circuit-reviews/"+queued.job.ID+"/decision", map[string]any{
			"circuit_revision_id": queued.persisted.Circuit.ID,
			"decision":            "reject",
			"note":                "cross-project",
			"reviewed_at":         "2026-05-23T09:00:00Z",
		}, http.StatusNotFound)
		if notFound["error"] == "" {
			t.Fatal("expected not found error body")
		}

		first := performJSONRequestExpectingStatus(t, env.handler, http.MethodPost, "/projects/prj-001/circuit-reviews/"+queued.job.ID+"/decision", map[string]any{
			"circuit_revision_id": queued.persisted.Circuit.ID,
			"decision":            "reject",
			"note":                "first",
			"reviewed_at":         "2026-05-23T09:00:00Z",
		}, http.StatusOK)
		if first["decision"] == nil {
			t.Fatal("expected first decision response")
		}

		second := performJSONRequestExpectingStatus(t, env.handler, http.MethodPost, "/projects/prj-001/circuit-reviews/"+queued.job.ID+"/decision", map[string]any{
			"circuit_revision_id": queued.persisted.Circuit.ID,
			"decision":            "reject",
			"note":                "second",
			"reviewed_at":         "2026-05-23T10:00:00Z",
		}, http.StatusConflict)
		if second["error"] == "" {
			t.Fatal("expected second decision conflict")
		}
	})
}

func newTestServer() http.Handler {
	return newTestServerWithIDs("ws-001", "prj-001", "prj-002", "job-001")
}

func newTestServerWithIDs(ids ...string) http.Handler {
	return server.New(server.Dependencies{
		Clock:       fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)},
		IDGenerator: &sequenceIDGenerator{ids: ids},
	})
}

type reviewHTTPEnv struct {
	handler  http.Handler
	jobs     *processing.Service
	repo     *circuit.MemoryRepository
	storage  storage.ObjectStorage
	circuits *circuit.Service
	clock    fixedClock
	ids      *sequenceIDGenerator
}

type seededReview struct {
	job       processing.Job
	persisted circuit.PersistOutput
}

func newReviewHTTPEnv() reviewHTTPEnv {
	return newReviewHTTPEnvWithIDs("job-001", "ext-001", "cir-001", "job-002", "ext-002", "cir-002", "job-003", "ext-003", "cir-003", "decision-001", "cir-approve-001")
}

func newReviewHTTPEnvWithIDs(ids ...string) reviewHTTPEnv {
	clock := fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)}
	idGen := &sequenceIDGenerator{ids: ids}
	jobRepo := processing.NewMemoryRepository()
	jobs := processing.NewService(jobRepo, clock, idGen)
	repo := circuit.NewMemoryRepository().WithProcessingRepository(jobRepo)
	objectStorage := server.NewMemoryObjectStorage()
	circuits := circuit.NewService(repo, objectStorage, clock, idGen, 0.8)
	handler := server.New(server.Dependencies{
		JobRepo:       jobRepo,
		JobSvc:        jobs,
		CircuitRepo:   repo,
		CircuitSvc:    circuits,
		ObjectStorage: objectStorage,
		Clock:         clock,
		IDGenerator:   idGen,
	})
	return reviewHTTPEnv{handler: handler, jobs: jobs, repo: repo, storage: objectStorage, circuits: circuits, clock: clock, ids: idGen}
}

func (e reviewHTTPEnv) seedJob(t *testing.T, projectID string, confidence float64, warnings []string) seededReview {
	t.Helper()
	job, err := e.jobs.CreateJob(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "ws-001", projectID, "up-"+projectID+"-001", 1)
	if err != nil {
		t.Fatalf("CreateJob error = %v", err)
	}
	job, err = e.jobs.Transition(httptest.NewRequest(http.MethodGet, "/", nil).Context(), job.ID, processing.JobStatusCreated, processing.JobStatusUploaded)
	if err != nil {
		t.Fatalf("transition to uploaded: %v", err)
	}
	job, err = e.jobs.Transition(httptest.NewRequest(http.MethodGet, "/", nil).Context(), job.ID, processing.JobStatusUploaded, processing.JobStatusExtracting)
	if err != nil {
		t.Fatalf("transition to extracting: %v", err)
	}
	persisted, err := e.circuits.Persist(httptest.NewRequest(http.MethodGet, "/", nil).Context(), circuit.PersistInput{
		WorkspaceID: "ws-001",
		ProjectID:   projectID,
		UploadID:    job.UploadID,
		Job:         job,
		Result:      circuit.ExtractionResult{Provider: "mock", Confidence: confidence, Warnings: warnings},
	})
	if err != nil {
		t.Fatalf("Persist error = %v", err)
	}
	job, err = e.jobs.MarkCompleted(httptest.NewRequest(http.MethodGet, "/", nil).Context(), job.ID, persisted.Status)
	if err != nil {
		t.Fatalf("MarkCompleted error = %v", err)
	}
	return seededReview{job: job, persisted: persisted}
}

func performJSONRequest(t *testing.T, handler http.Handler, method, path string, body any) map[string]any {
	t.Helper()
	return performJSONRequestExpectingStatus(t, handler, method, path, body, http.StatusCreated)
}

func performJSONRequestExpectingStatus(t *testing.T, handler http.Handler, method, path string, body any, wantStatus int) map[string]any {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	response := performRequest(t, handler, method, path, bytes.NewReader(payload))
	if response.Code != wantStatus {
		t.Fatalf("unexpected status %d", response.Code)
	}

	var decoded map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return decoded
}

func performRequest(t *testing.T, handler http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func performMultipartRequest(t *testing.T, handler http.Handler, method, path, fileName, contentType string, payload []byte) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	headers := textproto.MIMEHeader{}
	headers.Set("Content-Disposition", `form-data; name="file"; filename="`+fileName+`"`)
	headers.Set("Content-Type", contentType)
	part, err := writer.CreatePart(headers)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func normalizeBody(t *testing.T, input []byte) string {
	t.Helper()

	var value any
	if err := json.Unmarshal(input, &value); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	normalized, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("encode normalized body: %v", err)
	}

	return string(normalized) + "\n"
}

func assertGolden(t *testing.T, fileName, actual string) {
	t.Helper()

	path := filepath.Join("testdata", fileName)
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}

	if actual != string(expected) {
		t.Fatalf("golden mismatch for %s\nexpected:\n%s\nactual:\n%s", fileName, string(expected), actual)
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDGenerator struct {
	ids []string
	idx int
}

func (g *sequenceIDGenerator) NewID() string {
	if g.idx >= len(g.ids) {
		return platform.RandomIDGenerator{}.NewID()
	}
	id := g.ids[g.idx]
	g.idx++
	return id
}
