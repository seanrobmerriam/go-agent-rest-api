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
	toolRoot := os.Getenv("WORKSPACE_ROOT")
	if toolRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			logger.Error("failed to determine workspace root", "err", err)
			os.Exit(1)
		}
		toolRoot = cwd
	}

	builtin.Register(registry, builtin.Config{WorkspaceRoot: toolRoot})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	router := api.NewRouter(registry, jobStore, os.Getenv("API_KEY"))
	logger.Info("agent API listening", "addr", addr, "workspace_root", toolRoot)
	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}
