package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/msi/circuit-storys/backend/internal/config"
	"github.com/msi/circuit-storys/backend/internal/processing"
)

type stubRunner struct {
	err    error
	called bool
}

func (s *stubRunner) ProcessUntilEmpty(context.Context) error {
	s.called = true
	return s.err
}

func TestRunTreatsEmptyQueueAsSuccess(t *testing.T) {
	runner := &stubRunner{err: processing.ErrQueueEmpty}
	if err := run(context.Background(), runner); err != nil {
		t.Fatalf("run error = %v, want nil", err)
	}
	if !runner.called {
		t.Fatal("expected runner to be called")
	}
}

func TestRunReturnsProcessingError(t *testing.T) {
	wantErr := errors.New("boom")
	runner := &stubRunner{err: wantErr}
	if err := run(context.Background(), runner); !errors.Is(err, wantErr) {
		t.Fatalf("run error = %v, want %v", err, wantErr)
	}
}

func TestExecuteUsesConfiguredRunner(t *testing.T) {
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("REDIS_QUEUE_KEY", "jobs")
	t.Setenv("REVIEW_MIN_CONFIDENCE", "0.9")

	originalFactory := runnerFactory
	originalExec := runnerExec
	t.Cleanup(func() {
		runnerFactory = originalFactory
		runnerExec = originalExec
	})

	stub := &stubRunner{}
	runnerFactory = func(cfg config.Config) runner {
		if cfg.RedisAddr != "redis:6379" || cfg.RedisQueueKey != "jobs" || cfg.ReviewMinConfidence != 0.9 {
			t.Fatalf("unexpected config %+v", cfg)
		}
		return stub
	}
	runnerExec = func(ctx context.Context, got runner) error {
		if got != stub {
			t.Fatal("expected execute to use factory runner")
		}
		return nil
	}

	if err := execute(); err != nil {
		t.Fatalf("execute error = %v", err)
	}
}

func TestMainReturnsWhenExecuteSucceeds(t *testing.T) {
	originalFactory := runnerFactory
	originalExec := runnerExec
	t.Cleanup(func() {
		runnerFactory = originalFactory
		runnerExec = originalExec
	})

	runnerFactory = func(config.Config) runner { return &stubRunner{} }
	called := false
	runnerExec = func(context.Context, runner) error {
		called = true
		return nil
	}

	main()
	if !called {
		t.Fatal("expected main to execute runner")
	}
}

func TestBuildRunnerUsesConfiguredRedisAddress(t *testing.T) {
	runner := buildRunner(config.Config{RedisAddr: "127.0.0.1:0", RedisQueueKey: "jobs", ReviewMinConfidence: 0.9})
	err := run(context.Background(), runner)
	if err == nil {
		t.Fatal("expected dequeue error from configured redis address")
	}
	if !strings.Contains(err.Error(), "127.0.0.1:0") {
		t.Fatalf("error = %q, want redis address in error", err.Error())
	}
}
