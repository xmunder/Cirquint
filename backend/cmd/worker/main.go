package main

import (
	"context"
	"errors"
	"log"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/config"
	"github.com/msi/circuit-storys/backend/internal/platform"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/provider"
	"github.com/msi/circuit-storys/backend/internal/server"
	"github.com/msi/circuit-storys/backend/internal/uploads"
	"github.com/msi/circuit-storys/backend/internal/worker"
)

var (
	runnerFactory = buildRunner
	runnerExec    = run
)

type runner interface {
	ProcessUntilEmpty(context.Context) error
}

func buildRunner(cfg config.Config) runner {
	clock := platform.SystemClock{}
	idGen := platform.RandomIDGenerator{}
	queue := processing.NewRedisQueue(cfg.RedisAddr, cfg.RedisQueueKey)
	jobs := processing.NewService(processing.NewMemoryRepository(), clock, idGen).WithQueue(queue)
	uploadRepo := uploads.NewMemoryRepository()
	objectStorage := server.NewMemoryObjectStorage()
	circuitSvc := circuit.NewService(circuit.NewMemoryRepository(), objectStorage, clock, idGen, cfg.ReviewMinConfidence)
	return worker.NewRunner(jobs, uploadRepo, objectStorage, provider.UnconfiguredProvider{}, circuitSvc)
}

func run(ctx context.Context, runner runner) error {
	err := runner.ProcessUntilEmpty(ctx)
	if errors.Is(err, processing.ErrQueueEmpty) {
		return nil
	}
	return err
}

func execute() error {
	return runnerExec(context.Background(), runnerFactory(config.Load()))
}

func main() {
	if err := execute(); err != nil {
		log.Fatal(err)
	}
}
