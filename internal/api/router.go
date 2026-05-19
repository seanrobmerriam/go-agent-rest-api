package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/jobs"
	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func NewRouter(registry *tools.Registry, jobStore *jobs.Store, apiKey string) http.Handler {

	mux := http.NewServeMux()

	auth := requireAPIKey(apiKey)

	// GET /v1/tools — agent discovery
	mux.Handle("GET /v1/tools", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Respond(w, http.StatusOK, map[string]any{
			"tools": registry.List(),
		})
	})))

	// POST /v1/tools/{name} — tool invocation
	mux.Handle("POST /v1/tools/{name}", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")

		var input json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			// Allow empty body — treat as {}
			input = json.RawMessage("{}")
		}

		result, err := registry.Invoke(r.Context(), name, input)
		if err != nil {
			var notFound *tools.NotFoundError
			if errors.As(err, &notFound) {
				RespondErr(w, http.StatusNotFound, ErrNotFound, err.Error())
				return
			}
			RespondErr(w, http.StatusUnprocessableEntity, ErrToolError, err.Error())
			return
		}

		Respond(w, http.StatusOK, result)
	})))

	//POST /v1/jobs — start a background job
	mux.Handle("POST /v1/jobs", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Tool  string          `json:"tool"`
			Input json.RawMessage `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondErr(w, http.StatusBadRequest, ErrInvalidInput, "invalid request body")
			return
		}
		if body.Tool == "" {
			RespondErr(w, http.StatusBadRequest, ErrInvalidInput, "field 'tool' is required")
			return
		}
		if body.Input == nil {
			body.Input = json.RawMessage("{}")
		}

		job := jobStore.Create(body.Tool)

		jobStore.Run(r.Context(), job.ID, func(ctx context.Context) (any, error) {
			return registry.Invoke(ctx, body.Tool, body.Input)
		})

		Respond(w, http.StatusAccepted, job)
	})))

	// GET /v1/jobs/{id}
	mux.Handle("GET /v1/jobs/{id}", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		job, ok := jobStore.Get(id)
		if !ok {
			RespondErr(w, http.StatusNotFound, ErrNotFound, "job not found")
			return
		}
		Respond(w, http.StatusOK, job)
	})))

	// GET /v1/health — liveness probe (no auth needed)
	mux.HandleFunc("GET /v1/health", func(w http.ResponseWriter, r *http.Request) {
		Respond(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return mux
}
