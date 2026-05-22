package main

import (
	"log"
	"net/http"

	"github.com/msi/circuit-storys/backend/internal/config"
	"github.com/msi/circuit-storys/backend/internal/server"
)

func main() {
	cfg := config.Load()
	handler := server.New(server.Dependencies{})

	log.Printf("api listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, handler); err != nil {
		log.Fatal(err)
	}
}
