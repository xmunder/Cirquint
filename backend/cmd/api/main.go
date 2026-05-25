package main

import (
	"io"
	"log"
	"net/http"

	"github.com/msi/circuit-storys/backend/internal/config"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/server"
)

var listenAndServe = http.ListenAndServe

func newHandler(cfg config.Config) http.Handler {
	return server.New(server.Dependencies{
		Queue: processing.NewRedisQueue(cfg.RedisAddr, cfg.RedisQueueKey),
	})
}

func run(cfg config.Config, serve func(string, http.Handler) error, logger *log.Logger) error {
	handler := newHandler(cfg)
	logger.Printf("api listening on %s", cfg.HTTPAddr)
	return serve(cfg.HTTPAddr, handler)
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
