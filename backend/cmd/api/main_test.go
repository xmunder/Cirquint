package main

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msi/circuit-storys/backend/internal/config"
)

func TestRunPassesAddressAndHandler(t *testing.T) {
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
