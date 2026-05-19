// External HTTP call — shows context cancellation propagation and an optional field with a default.

package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func registerHTTPGet(r *tools.Registry) {
	r.Register(tools.ToolDef{
		Name:        "http_get",
		Description: "Fetches a URL via HTTP GET and returns the status code and response body. Body is truncated to 4096 characters.",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"url": {
					Type:        "string",
					Description: "The URL to fetch",
				},
				"timeout_seconds": {
					Type:        "integer",
					Description: "Request timeout in seconds. Defaults to 10 if omitted.",
				},
			},
			Required: []string{"url"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var params struct {
			URL            string `json:"url"`
			TimeoutSeconds int    `json:"timeout_seconds"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}

		// Optional field with default
		if params.TimeoutSeconds <= 0 {
			params.TimeoutSeconds = 10
		}

		ctx, cancel := context.WithTimeout(ctx, time.Duration(params.TimeoutSeconds)*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		return map[string]any{
			"status_code": resp.StatusCode,
			"body":        string(body),
			"truncated":   len(body) == 4096,
		}, nil
	})
}
