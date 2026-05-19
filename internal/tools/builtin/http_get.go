// External HTTP call — shows context cancellation propagation and an optional field with a default.

package builtin

import (
	"context"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
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

		return executeHTTPRequest(ctx, httpRequestParams{
			Method:         http.MethodGet,
			URL:            params.URL,
			TimeoutSeconds: params.TimeoutSeconds,
		})
	})
}

func registerHTTPRequest(r *tools.Registry) {
	r.Register(tools.ToolDef{
		Name:        "http_request",
		Description: "Makes an outbound HTTP request with a controlled method, headers, timeout, and optional body. Response body is truncated to 4096 characters.",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"method": {
					Type:        "string",
					Description: "HTTP method to use.",
					Enum:        []any{"GET", "POST", "PUT", "PATCH", "DELETE"},
				},
				"url": {
					Type:        "string",
					Description: "The URL to request.",
				},
				"headers": {
					Type:        "object",
					Description: "Optional string headers to send with the request.",
				},
				"body": {
					Type:        "string",
					Description: "Optional request body for non-GET methods.",
				},
				"timeout_seconds": {
					Type:        "integer",
					Description: "Request timeout in seconds. Defaults to 10 if omitted. Maximum 30.",
				},
			},
			Required: []string{"method", "url"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var params struct {
			Method         string            `json:"method"`
			URL            string            `json:"url"`
			Headers        map[string]string `json:"headers"`
			Body           string            `json:"body"`
			TimeoutSeconds int               `json:"timeout_seconds"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}

		return executeHTTPRequest(ctx, httpRequestParams{
			Method:         params.Method,
			URL:            params.URL,
			Headers:        params.Headers,
			Body:           params.Body,
			TimeoutSeconds: params.TimeoutSeconds,
		})
	})
}

type httpRequestParams struct {
	Method         string
	URL            string
	Headers        map[string]string
	Body           string
	TimeoutSeconds int
}

func executeHTTPRequest(ctx context.Context, params httpRequestParams) (any, error) {
	if params.TimeoutSeconds <= 0 {
		params.TimeoutSeconds = 10
	}
	if params.TimeoutSeconds > 30 {
		params.TimeoutSeconds = 30
	}

	method := params.Method
	if method == "" {
		method = http.MethodGet
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(params.TimeoutSeconds)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, params.URL, bytes.NewBufferString(params.Body))
	if err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	for key, value := range params.Headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	const maxBodyBytes = 4096
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	headerNames := make([]string, 0, len(resp.Header))
	for name := range resp.Header {
		headerNames = append(headerNames, name)
	}
	sort.Strings(headerNames)

	headers := make(map[string]string, len(resp.Header))
	for _, name := range headerNames {
		headers[name] = resp.Header.Get(name)
	}

	return map[string]any{
		"status_code": resp.StatusCode,
		"body":        string(body),
		"truncated":   len(body) == maxBodyBytes,
		"headers":     headers,
		"method":      method,
		"url":         params.URL,
	}, nil
}
