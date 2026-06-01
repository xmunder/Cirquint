package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/config"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/provider"
	"github.com/msi/circuit-storys/backend/internal/server"
	"github.com/msi/circuit-storys/backend/internal/storage"
	"github.com/msi/circuit-storys/backend/internal/testpostgres"
	"github.com/msi/circuit-storys/backend/internal/uploads"
	workerpkg "github.com/msi/circuit-storys/backend/internal/worker"
	"github.com/msi/circuit-storys/backend/internal/workspace"
)

func TestRunPassesAddressAndHandler(t *testing.T) {
	stubOpenDatabase(t)
	cfg := config.Config{
		HTTPAddr:      ":9090",
		RedisAddr:     "redis:6379",
		RedisQueueKey: "jobs",
		DatabaseURL:   "postgres://app:secret@db.internal:5432/cirquint",
		ObjectStorage: config.ObjectStorageConfig{
			Bucket:    "cirquint-dev",
			Endpoint:  "https://r2.example.com",
			AccessKey: "access-key",
			SecretKey: "secret-key",
		},
	}
	var gotAddr string
	var gotHandler http.Handler
	var logs bytes.Buffer
	originalFactory := handlerFactory
	t.Cleanup(func() { handlerFactory = originalFactory })
	handlerFactory = func(cfg config.Config) (http.Handler, error) {
		return newHandler(cfg)
	}

	err := run(cfg, func(addr string, handler http.Handler) error {
		gotAddr = addr
		gotHandler = handler
		return nil
	}, log.New(&logs, "", 0))
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if gotAddr != cfg.HTTPAddr {
		t.Fatalf("serve addr = %q, want %q", gotAddr, cfg.HTTPAddr)
	}
	assertHandlerReturnsNotFound(t, gotHandler)
	if logs.Len() == 0 {
		t.Fatal("expected listen log output")
	}
}

func TestRunReturnsServeError(t *testing.T) {
	stubOpenDatabase(t)
	wantErr := errors.New("listen failed")
	err := run(config.Config{
		HTTPAddr:    ":8080",
		DatabaseURL: "postgres://app:secret@db.internal:5432/cirquint",
		ObjectStorage: config.ObjectStorageConfig{
			Bucket:    "cirquint-dev",
			Endpoint:  "https://r2.example.com",
			AccessKey: "access-key",
			SecretKey: "secret-key",
		},
	}, func(string, http.Handler) error {
		return wantErr
	}, log.New(&bytes.Buffer{}, "", 0))
	if !errors.Is(err, wantErr) {
		t.Fatalf("run error = %v, want %v", err, wantErr)
	}
}

func TestExecuteUsesLoadedConfig(t *testing.T) {
	stubOpenDatabase(t)
	t.Setenv("HTTP_ADDR", ":9191")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("REDIS_QUEUE_KEY", "jobs")
	t.Setenv("DATABASE_URL", "postgres://app:secret@db.internal:5432/cirquint")
	t.Setenv("OBJECT_STORAGE_BUCKET", "cirquint-dev")
	t.Setenv("OBJECT_STORAGE_ENDPOINT", "https://r2.example.com")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY", "access-key")
	t.Setenv("OBJECT_STORAGE_SECRET_KEY", "secret-key")

	original := listenAndServe
	originalFactory := handlerFactory
	t.Cleanup(func() { listenAndServe = original })
	t.Cleanup(func() { handlerFactory = originalFactory })

	var gotAddr string
	var gotConfig config.Config
	handlerFactory = func(cfg config.Config) (http.Handler, error) {
		gotConfig = cfg
		return newHandler(cfg)
	}
	listenAndServe = func(addr string, handler http.Handler) error {
		gotAddr = addr
		assertHandlerReturnsNotFound(t, handler)
		return nil
	}

	if err := execute(); err != nil {
		t.Fatalf("execute error = %v", err)
	}
	if gotAddr != ":9191" {
		t.Fatalf("addr = %q, want %q", gotAddr, ":9191")
	}
	if gotConfig.DatabaseURL != "postgres://app:secret@db.internal:5432/cirquint" {
		t.Fatalf("DatabaseURL = %q, want loaded env", gotConfig.DatabaseURL)
	}
	if gotConfig.ObjectStorage.Bucket != "cirquint-dev" {
		t.Fatalf("ObjectStorage.Bucket = %q, want loaded env", gotConfig.ObjectStorage.Bucket)
	}
}

func TestExecuteReturnsListenError(t *testing.T) {
	stubOpenDatabase(t)
	t.Setenv("DATABASE_URL", "postgres://app:secret@db.internal:5432/cirquint")
	t.Setenv("OBJECT_STORAGE_BUCKET", "cirquint-dev")
	t.Setenv("OBJECT_STORAGE_ENDPOINT", "https://r2.example.com")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY", "access-key")
	t.Setenv("OBJECT_STORAGE_SECRET_KEY", "secret-key")
	wantErr := errors.New("serve failed")
	original := listenAndServe
	originalFactory := handlerFactory
	t.Cleanup(func() { listenAndServe = original })
	t.Cleanup(func() { handlerFactory = originalFactory })
	handlerFactory = func(cfg config.Config) (http.Handler, error) {
		return newHandler(cfg)
	}
	listenAndServe = func(string, http.Handler) error { return wantErr }

	if err := execute(); !errors.Is(err, wantErr) {
		t.Fatalf("execute error = %v, want %v", err, wantErr)
	}
}

func TestMainReturnsWhenServeSucceeds(t *testing.T) {
	stubOpenDatabase(t)
	t.Setenv("HTTP_ADDR", ":9292")
	t.Setenv("DATABASE_URL", "postgres://app:secret@db.internal:5432/cirquint")
	t.Setenv("OBJECT_STORAGE_BUCKET", "cirquint-dev")
	t.Setenv("OBJECT_STORAGE_ENDPOINT", "https://r2.example.com")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY", "access-key")
	t.Setenv("OBJECT_STORAGE_SECRET_KEY", "secret-key")
	original := listenAndServe
	originalFactory := handlerFactory
	t.Cleanup(func() { listenAndServe = original })
	t.Cleanup(func() { handlerFactory = originalFactory })
	handlerFactory = func(cfg config.Config) (http.Handler, error) {
		return newHandler(cfg)
	}
	called := false
	listenAndServe = func(string, http.Handler) error {
		called = true
		return nil
	}

	main()
	if !called {
		t.Fatal("expected main to invoke listenAndServe")
	}
}

func TestNewHandlerRejectsMissingSharedRuntimeConfig(t *testing.T) {
	_, err := newHandler(config.Config{RedisAddr: "redis:6379", RedisQueueKey: "jobs"})
	if !errors.Is(err, config.ErrDatabaseURLRequired) {
		t.Fatalf("newHandler error = %v, want %v", err, config.ErrDatabaseURLRequired)
	}
}

func TestNewHandlerUsesPostgresRepositoriesWhenDatabaseConfigured(t *testing.T) {
	originalOpen := openDatabase
	originalServerFactory := serverFactory
	t.Cleanup(func() {
		openDatabase = originalOpen
		serverFactory = originalServerFactory
	})

	var gotURL string
	openDatabase = func(databaseURL string) (*sql.DB, error) {
		gotURL = databaseURL
		return nil, nil
	}

	var captured server.Dependencies
	serverFactory = func(deps server.Dependencies) http.Handler {
		captured = deps
		return http.NotFoundHandler()
	}
	storageRoot := t.TempDir()

	handler, err := newHandler(config.Config{
		RedisAddr:     "redis:6379",
		RedisQueueKey: "jobs",
		DatabaseURL:   "postgres://app:secret@db.internal:5432/cirquint",
		ObjectStorage: config.ObjectStorageConfig{
			Bucket:    "cirquint-dev",
			Endpoint:  "https://r2.example.com",
			AccessKey: "access-key",
			Path:      storageRoot,
			SecretKey: "secret-key",
		},
	})
	if err != nil {
		t.Fatalf("newHandler error = %v", err)
	}
	if gotURL != "postgres://app:secret@db.internal:5432/cirquint" {
		t.Fatalf("openDatabase url = %q", gotURL)
	}
	if _, ok := captured.JobRepo.(*processing.PostgresRepository); !ok {
		t.Fatalf("JobRepo type = %T, want *processing.PostgresRepository", captured.JobRepo)
	}
	if _, ok := captured.CircuitRepo.(*circuit.PostgresRepository); !ok {
		t.Fatalf("CircuitRepo type = %T, want *circuit.PostgresRepository", captured.CircuitRepo)
	}
	if _, ok := captured.WorkspaceRepo.(*workspace.PostgresRepository); !ok {
		t.Fatalf("WorkspaceRepo type = %T, want *workspace.PostgresRepository", captured.WorkspaceRepo)
	}
	if _, ok := captured.UploadRepo.(*uploads.PostgresRepository); !ok {
		t.Fatalf("UploadRepo type = %T, want *uploads.PostgresRepository", captured.UploadRepo)
	}
	if captured.ObjectStorage == nil {
		t.Fatal("expected shared object storage dependency")
	}
	if _, err := captured.ObjectStorage.Put(context.Background(), "proof/object.txt", bytes.NewBufferString("shared"), storage.ObjectMeta{}); err != nil {
		t.Fatalf("ObjectStorage.Put error = %v", err)
	}
	reader := storage.NewFilesystemObjectStorage(storageRoot)
	body, err := reader.Get(context.Background(), "proof/object.txt")
	if err != nil {
		t.Fatalf("filesystem reader Get error = %v", err)
	}
	defer body.Close()
	assertHandlerReturnsNotFound(t, handler)
}

func TestUploadRouteAndWorkerUsePostgresWorkspaceAndUploadRepositories(t *testing.T) {
	db := openAPITestPostgres(t)
	storageRoot := t.TempDir()
	apiStorage := storage.NewFilesystemObjectStorage(storageRoot)
	workerStorage := storage.NewFilesystemObjectStorage(storageRoot)
	queue := &recordingQueue{}
	clock := fixedClock{now: time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)}
	ids := &sequenceIDs{ids: []string{"ws-001", "prj-001", "up-001", "job-001", "ext-001", "cir-001"}}

	handler := server.New(server.Dependencies{
		WorkspaceRepo: workspace.NewPostgresRepository(db),
		UploadRepo:    uploads.NewPostgresRepository(db),
		JobRepo:       processing.NewPostgresRepository(db),
		CircuitRepo:   circuit.NewPostgresRepository(db),
		Queue:         queue,
		ObjectStorage: apiStorage,
		Clock:         clock,
		IDGenerator:   ids,
	})

	workspaceID := decodeIDResponse(t, performJSONRequest(t, handler, http.MethodPost, "/workspaces", map[string]string{"name": "Workspace Alpha"}))
	projectID := decodeIDResponse(t, performJSONRequest(t, handler, http.MethodPost, "/workspaces/"+workspaceID+"/projects", map[string]string{"name": "Project One"}))

	uploadResponse := performMultipartRequest(t, handler, http.MethodPost, "/projects/"+projectID+"/uploads", "diagram.png", "image/png", []byte("png-data"))
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, want %d", uploadResponse.Code, http.StatusCreated)
	}

	if len(queue.payloads) != 1 {
		t.Fatalf("queued payloads = %d, want 1", len(queue.payloads))
	}
	uploadRepo := uploads.NewPostgresRepository(db)
	storedUpload, err := uploadRepo.GetUpload(context.Background(), "up-001")
	if err != nil {
		t.Fatalf("GetUpload error = %v", err)
	}
	if storedUpload.Status != uploads.UploadStatusUploaded {
		t.Fatalf("upload status = %s, want %s", storedUpload.Status, uploads.UploadStatusUploaded)
	}

	runner := workerpkg.NewRunner(
		processing.NewService(processing.NewPostgresRepository(db), clock, &sequenceIDs{}).WithQueue(queue),
		uploadRepo,
		workerStorage,
		provider.StaticProvider{Result: circuit.ExtractionResult{Provider: "mock", Confidence: 0.4, Warnings: []string{"ambiguous-node-label"}}},
		circuit.NewService(circuit.NewPostgresRepository(db), workerStorage, clock, &sequenceIDs{ids: []string{"ext-001", "cir-001"}}, 0.8),
	)
	if err := runner.ProcessUntilEmpty(context.Background()); err != nil {
		t.Fatalf("ProcessUntilEmpty error = %v", err)
	}

	job, err := processing.NewPostgresRepository(db).GetJob(context.Background(), "job-001")
	if err != nil {
		t.Fatalf("GetJob error = %v", err)
	}
	if job.Status != processing.JobStatusNeedsReview {
		t.Fatalf("job status = %s, want %s", job.Status, processing.JobStatusNeedsReview)
	}
}

func assertHandlerReturnsNotFound(t *testing.T, handler http.Handler) {
	t.Helper()
	if handler == nil {
		t.Fatal("handler must not be nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
	}
	if !strings.Contains(resp.Body.String(), "not found") {
		t.Fatalf("body = %q, want error containing %q", resp.Body.String(), "not found")
	}
}

func stubOpenDatabase(t *testing.T) {
	t.Helper()
	original := openDatabase
	t.Cleanup(func() { openDatabase = original })
	openDatabase = func(string) (*sql.DB, error) { return nil, nil }
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type sequenceIDs struct {
	ids []string
	idx int
}

func (g *sequenceIDs) NewID() string {
	if g.idx >= len(g.ids) {
		return "fallback-id"
	}
	id := g.ids[g.idx]
	g.idx++
	return id
}

type recordingQueue struct {
	payloads []processing.Payload
	idx      int
}

func (q *recordingQueue) Enqueue(_ context.Context, payload processing.Payload) error {
	q.payloads = append(q.payloads, payload)
	return nil
}

func (q *recordingQueue) Dequeue(context.Context) (processing.Payload, error) {
	if q.idx >= len(q.payloads) {
		return processing.Payload{}, processing.ErrQueueEmpty
	}
	payload := q.payloads[q.idx]
	q.idx++
	return payload, nil
}

func openAPITestPostgres(t *testing.T) *sql.DB {
	return testpostgres.Open(t, "cmd/api")
}

func performJSONRequest(t *testing.T, handler http.Handler, method, path string, body any) map[string]any {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return decoded
}

func decodeIDResponse(t *testing.T, body map[string]any) string {
	t.Helper()
	id, _ := body["id"].(string)
	if id == "" {
		t.Fatal("expected id in response")
	}
	return id
}

func performMultipartRequest(t *testing.T, handler http.Handler, method, path, filename, contentType string, payload []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	headers := textproto.MIMEHeader{}
	headers.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	headers.Set("Content-Type", contentType)
	part, err := writer.CreatePart(headers)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(payload)); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}
