package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/api"
	"github.com/seanrobmerriam/go-agent-rest-api/internal/jobs"
	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools/builtin"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	registry := tools.NewRegistry()
	jobStore := jobs.NewStore()

	builtin.Register(registry) // ← replaces the inline echo registration

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	router := api.NewRouter(registry, jobStore, os.Getenv("API_KEY"))
	logger.Info("agent API listening", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}
