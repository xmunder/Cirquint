package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"strings"

	_ "github.com/lib/pq"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/config"
	"github.com/msi/circuit-storys/backend/internal/platform"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/provider"
	"github.com/msi/circuit-storys/backend/internal/server"
	"github.com/msi/circuit-storys/backend/internal/storage"
	"github.com/msi/circuit-storys/backend/internal/uploads"
	"github.com/msi/circuit-storys/backend/internal/worker"
)

var (
	runnerFactory = buildRunner
	runnerExec    = run
	openDatabase  = openPostgresDatabase
)

type runner interface {
	ProcessUntilEmpty(context.Context) error
}

type managedRunner struct {
	runner
	closer io.Closer
}

func (r managedRunner) Close() error {
	if r.closer == nil {
		return nil
	}
	return r.closer.Close()
}

func buildRunner(cfg config.Config) (runner, error) {
	return buildRuntimeRunner(
		cfg,
		nil,
		nil,
		provider.UnconfiguredProvider{},
		processing.NewRedisQueue(cfg.RedisAddr, cfg.RedisQueueKey),
	)
}

func buildRuntimeRunner(cfg config.Config, uploadRepo uploads.Repository, objectStorage storage.ObjectStorage, extractionProvider provider.CircuitExtractionProvider, queue processing.Queue) (runner, error) {
	if err := cfg.ValidateSharedRuntime(); err != nil {
		return nil, err
	}
	db, err := openDatabase(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	clock := platform.SystemClock{}
	idGen := platform.RandomIDGenerator{}
	if uploadRepo == nil {
		uploadRepo = uploads.NewPostgresRepository(db)
	}
	if objectStorage == nil {
		objectStorage = buildObjectStorage(cfg)
	}
	jobs := processing.NewService(processing.NewPostgresRepository(db), clock, idGen).WithQueue(queue)
	circuitSvc := circuit.NewService(circuit.NewPostgresRepository(db), objectStorage, clock, idGen, cfg.ReviewMinConfidence)
	return managedRunner{runner: worker.NewRunner(jobs, uploadRepo, objectStorage, extractionProvider, circuitSvc), closer: db}, nil
}

func buildObjectStorage(cfg config.Config) storage.ObjectStorage {
	if strings.TrimSpace(cfg.ObjectStorage.Path) != "" {
		return storage.NewFilesystemObjectStorage(cfg.ObjectStorage.Path)
	}
	return server.NewMemoryObjectStorage()
}

func run(ctx context.Context, runner runner) error {
	err := runner.ProcessUntilEmpty(ctx)
	if errors.Is(err, processing.ErrQueueEmpty) {
		return nil
	}
	return err
}

func execute() error {
	runner, err := runnerFactory(config.Load())
	if err != nil {
		return err
	}
	if closer, ok := runner.(io.Closer); ok {
		defer closer.Close()
	}
	return runnerExec(context.Background(), runner)
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

func main() {
	if err := execute(); err != nil {
		log.Fatal(err)
	}
}
