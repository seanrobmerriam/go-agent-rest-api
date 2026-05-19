package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/api"
	"github.com/seanrobmerriam/go-agent-rest-api/internal/jobs"
	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	registry := tools.NewRegistry()
	jobStore := jobs.NewStore()
	router := api.NewRouter(registry, jobStore, os.Getenv("API_KEY"))

	// Register your tools here
	registry.Register(tools.ToolDef{
		Name:        "echo",
		Description: "Echoes the input message back. Useful for testing connectivity.",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"message": {Type: "string", Description: "The text to echo"},
			},
			Required: []string{"message"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var params struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, err
		}
		return map[string]string{"echo": params.Message}, nil
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	logger.Info("agent API listening", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}
