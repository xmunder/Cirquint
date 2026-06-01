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

	"github.com/msi/circuit-storys/backend/internal/platform"
	"github.com/msi/circuit-storys/backend/internal/server"
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

func newTestServer() http.Handler {
	return newTestServerWithIDs("ws-001", "prj-001", "prj-002", "job-001")
}

func newTestServerWithIDs(ids ...string) http.Handler {
	return server.New(server.Dependencies{
		Clock:       fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)},
		IDGenerator: &sequenceIDGenerator{ids: ids},
	})
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
