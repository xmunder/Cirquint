package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

	response := performRequest(t, handler, http.MethodPost, "/projects/missing-project/uploads", bytes.NewBufferString(`{}`))
	if response.Code != http.StatusNotFound {
		t.Fatalf("unexpected status %d", response.Code)
	}

	actual := normalizeBody(t, response.Body.Bytes())
	assertGolden(t, "upload_invalid_project.golden", actual)
}

func newTestServer() http.Handler {
	return server.New(server.Dependencies{
		Clock:       fixedClock{now: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)},
		IDGenerator: &sequenceIDGenerator{ids: []string{"ws-001", "prj-001", "prj-002"}},
	})
}

func performJSONRequest(t *testing.T, handler http.Handler, method, path string, body any) map[string]any {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	response := performRequest(t, handler, method, path, bytes.NewReader(payload))
	if response.Code != http.StatusCreated {
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
