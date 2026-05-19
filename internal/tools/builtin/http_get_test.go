package builtin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func TestHTTPTools(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Seen-Method", r.Method)
		w.Header().Set("X-Seen-Header", r.Header.Get("X-Test"))
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write([]byte("echo:" + string(body)))
	}))
	defer server.Close()

	registry := tools.NewRegistry()
	registerHTTPGet(registry)
	registerHTTPRequest(registry)

	tests := []struct {
		name        string
		tool        string
		input       string
		wantErrPart string
		assert      func(t *testing.T, result any)
	}{
		{
			name:  "http_get success",
			tool:  "http_get",
			input: `{"url":"` + server.URL + `"}`,
			assert: func(t *testing.T, result any) {
				t.Helper()
				payload := mustMap(t, result)
				if payload["status_code"] != http.StatusOK {
					t.Fatalf("unexpected status code: %#v", payload["status_code"])
				}
				if payload["method"] != http.MethodGet {
					t.Fatalf("unexpected method: %#v", payload["method"])
				}
			},
		},
		{
			name:  "http_request post with headers and body",
			tool:  "http_request",
			input: `{"method":"POST","url":"` + server.URL + `","headers":{"X-Test":"abc"},"body":"payload"}`,
			assert: func(t *testing.T, result any) {
				t.Helper()
				payload := mustMap(t, result)
				if payload["method"] != http.MethodPost {
					t.Fatalf("unexpected method: %#v", payload["method"])
				}
				body, _ := payload["body"].(string)
				if body != "echo:payload" {
					t.Fatalf("unexpected body: %q", body)
				}
				headers, ok := payload["headers"].(map[string]string)
				if !ok {
					t.Fatalf("headers not object: %T", payload["headers"])
				}
				if headers["X-Seen-Header"] != "abc" {
					t.Fatalf("expected server to see X-Test header, got %#v", headers["X-Seen-Header"])
				}
			},
		},
		{
			name:        "http_request invalid url",
			tool:        "http_request",
			input:       `{"method":"GET","url":":// bad"}`,
			wantErrPart: "invalid request",
		},
		{
			name:        "http_get missing url hits validation",
			tool:        "http_get",
			input:       `{}`,
			wantErrPart: `field "url" is required`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result, err := registry.Invoke(context.Background(), tc.tool, json.RawMessage(tc.input))

			if tc.wantErrPart != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrPart)
				}
				if !strings.Contains(err.Error(), tc.wantErrPart) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErrPart, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.assert != nil {
				tc.assert(t, result)
			}
		})
	}
}
