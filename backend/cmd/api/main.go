package main

import (
	"context"
	"database/sql"
	"io"
	"log"
	"net/http"
	"strings"

	_ "github.com/lib/pq"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/config"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/server"
	"github.com/msi/circuit-storys/backend/internal/storage"
	"github.com/msi/circuit-storys/backend/internal/uploads"
	"github.com/msi/circuit-storys/backend/internal/workspace"
)

var listenAndServe = http.ListenAndServe
var handlerFactory = newHandler
var openDatabase = openPostgresDatabase
var serverFactory = server.New

type managedHandler struct {
	http.Handler
	closer io.Closer
}

func (h managedHandler) Close() error {
	if h.closer == nil {
		return nil
	}
	return h.closer.Close()
}

func newHandler(cfg config.Config) (http.Handler, error) {
	deps, db, err := buildServerDependencies(cfg)
	if err != nil {
		return nil, err
	}

	handler := serverFactory(deps)
	if db == nil {
		return handler, nil
	}
	return managedHandler{Handler: handler, closer: db}, nil
}

func run(cfg config.Config, serve func(string, http.Handler) error, logger *log.Logger) error {
	handler, err := handlerFactory(cfg)
	if err != nil {
		return err
	}
	if closer, ok := handler.(io.Closer); ok {
		defer closer.Close()
	}
	logger.Printf("api listening on %s", cfg.HTTPAddr)
	return serve(cfg.HTTPAddr, handler)
}

func buildServerDependencies(cfg config.Config) (server.Dependencies, *sql.DB, error) {
	if err := cfg.ValidateSharedRuntime(); err != nil {
		return server.Dependencies{}, nil, err
	}

	db, err := openDatabase(cfg.DatabaseURL)
	if err != nil {
		return server.Dependencies{}, nil, err
	}

	return server.Dependencies{
		WorkspaceRepo: workspace.NewPostgresRepository(db),
		UploadRepo:    uploads.NewPostgresRepository(db),
		JobRepo:       processing.NewPostgresRepository(db),
		CircuitRepo:   circuit.NewPostgresRepository(db),
		Queue:         processing.NewRedisQueue(cfg.RedisAddr, cfg.RedisQueueKey),
		ObjectStorage: buildObjectStorage(cfg),
	}, db, nil
}

func buildObjectStorage(cfg config.Config) storage.ObjectStorage {
	if strings.TrimSpace(cfg.ObjectStorage.Path) != "" {
		return storage.NewFilesystemObjectStorage(cfg.ObjectStorage.Path)
	}
	return server.NewMemoryObjectStorage()
}

func openPostgresDatabase(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func execute() error {
	logger := log.New(io.Discard, "", 0)
	return run(config.Load(), listenAndServe, logger)
}

func main() {
	if err := execute(); err != nil {
		log.Fatal(err)
	}
}
