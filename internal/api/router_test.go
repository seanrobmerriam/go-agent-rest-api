package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/jobs"
	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func TestRouterToolInvocationAndValidation(t *testing.T) {
	t.Parallel()

	router := newTestRouter("")

	tests := []struct {
		name         string
		method       string
		path         string
		body         string
		wantStatus   int
		wantErrCode  string
		wantDataPath string
	}{
		{
			name:         "health endpoint",
			method:       http.MethodGet,
			path:         "/v1/health",
			wantStatus:   http.StatusOK,
			wantDataPath: "status",
		},
		{
			name:         "invoke tool success",
			method:       http.MethodPost,
			path:         "/v1/tools/sum",
			body:         `{"a":1,"b":2}`,
			wantStatus:   http.StatusOK,
			wantDataPath: "sum",
		},
		{
			name:        "tool input malformed json",
			method:      http.MethodPost,
			path:        "/v1/tools/sum",
			body:        `{"a":`,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: ErrInvalidInput,
		},
		{
			name:        "tool input fails required fields",
			method:      http.MethodPost,
			path:        "/v1/tools/sum",
			body:        `{"a":1}`,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: ErrInvalidInput,
		},
		{
			name:        "tool input rejects unknown fields",
			method:      http.MethodPost,
			path:        "/v1/tools/sum",
			body:        `{"a":1,"b":2,"extra":true}`,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: ErrInvalidInput,
		},
		{
			name:        "unknown tool",
			method:      http.MethodPost,
			path:        "/v1/tools/missing",
			body:        `{}`,
			wantStatus:  http.StatusNotFound,
			wantErrCode: ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp := performJSONRequest(t, router, tc.method, tc.path, tc.body, "")
			assertStatus(t, resp, tc.wantStatus)

			env := decodeEnvelope(t, resp)
			if tc.wantErrCode != "" {
				if env.Error == nil {
					t.Fatalf("expected error with code %q, got nil", tc.wantErrCode)
				}
				if env.Error.Code != tc.wantErrCode {
					t.Fatalf("expected error code %q, got %q", tc.wantErrCode, env.Error.Code)
				}
			}

			if tc.wantDataPath != "" {
				data, ok := env.Data.(map[string]any)
				if !ok {
					t.Fatalf("expected object data, got %T", env.Data)
				}
				if _, ok := data[tc.wantDataPath]; !ok {
					t.Fatalf("expected data field %q", tc.wantDataPath)
				}
			}
		})
	}
}

func TestRouterJobsAndLookup(t *testing.T) {
	t.Parallel()

	router := newTestRouter("")

	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantErrCode string
	}{
		{
			name:       "create job success",
			body:       `{"tool":"sum","input":{"a":1,"b":2}}`,
			wantStatus: http.StatusAccepted,
		},
		{
			name:        "create job malformed body",
			body:        `{"tool":`,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: ErrInvalidInput,
		},
		{
			name:        "create job missing tool",
			body:        `{"input":{"a":1,"b":2}}`,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: ErrInvalidInput,
		},
		{
			name:        "create job invalid tool input",
			body:        `{"tool":"sum","input":{"a":1}}`,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: ErrInvalidInput,
		},
	}

	var createdID string
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp := performJSONRequest(t, router, http.MethodPost, "/v1/jobs", tc.body, "")
			assertStatus(t, resp, tc.wantStatus)

			env := decodeEnvelope(t, resp)
			if tc.wantErrCode != "" {
				if env.Error == nil || env.Error.Code != tc.wantErrCode {
					t.Fatalf("expected error code %q, got %+v", tc.wantErrCode, env.Error)
				}
				return
			}

			data := mustDataObject(t, env)
			id, ok := data["id"].(string)
			if !ok || id == "" {
				t.Fatalf("expected job id in response")
			}
			createdID = id
		})
	}

	if createdID == "" {
		t.Fatalf("expected created job id")
	}

	deadline := time.Now().Add(1 * time.Second)
	for {
		resp := performJSONRequest(t, router, http.MethodGet, "/v1/jobs/"+createdID, "", "")
		assertStatus(t, resp, http.StatusOK)
		env := decodeEnvelope(t, resp)
		data := mustDataObject(t, env)
		status, _ := data["status"].(string)
		if status == string(jobs.StatusDone) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for job completion, last status=%q", status)
		}
	}

	notFoundResp := performJSONRequest(t, router, http.MethodGet, "/v1/jobs/does-not-exist", "", "")
	assertStatus(t, notFoundResp, http.StatusNotFound)
	notFoundEnv := decodeEnvelope(t, notFoundResp)
	if notFoundEnv.Error == nil || notFoundEnv.Error.Code != ErrNotFound {
		t.Fatalf("expected NOT_FOUND for missing job")
	}
}

func TestRouterAuthModes(t *testing.T) {
	t.Parallel()

	protectedPath := "/v1/tools"

	t.Run("protected endpoint rejects missing auth when key set", func(t *testing.T) {
		router := newTestRouter("topsecret")
		resp := performJSONRequest(t, router, http.MethodGet, protectedPath, "", "")
		assertStatus(t, resp, http.StatusUnauthorized)
		env := decodeEnvelope(t, resp)
		if env.Error == nil || env.Error.Code != ErrUnauthorized {
			t.Fatalf("expected unauthorized error, got %+v", env.Error)
		}
	})

	t.Run("protected endpoint allows valid auth", func(t *testing.T) {
		router := newTestRouter("topsecret")
		resp := performJSONRequest(t, router, http.MethodGet, protectedPath, "", "Bearer topsecret")
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("health remains public when key set", func(t *testing.T) {
		router := newTestRouter("topsecret")
		resp := performJSONRequest(t, router, http.MethodGet, "/v1/health", "", "")
		assertStatus(t, resp, http.StatusOK)
	})
}

func newTestRouter(apiKey string) http.Handler {
	registry := tools.NewRegistry()
	registry.Register(tools.ToolDef{
		Name:        "sum",
		Description: "sum two integers",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"a": {Type: "integer"},
				"b": {Type: "integer"},
			},
			Required: []string{"a", "b"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var req struct {
			A int `json:"a"`
			B int `json:"b"`
		}
		_ = json.Unmarshal(input, &req)
		return map[string]int{"sum": req.A + req.B}, nil
	})

	jobStore := jobs.NewStore()
	return NewRouter(registry, jobStore, apiKey)
}

func performJSONRequest(t *testing.T, handler http.Handler, method, path, body, authHeader string) *httptest.ResponseRecorder {
	t.Helper()

	requestBody := bytes.NewBufferString(body)
	req := httptest.NewRequest(method, path, requestBody)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}

func assertStatus(t *testing.T, resp *httptest.ResponseRecorder, want int) {
	t.Helper()
	if resp.Code != want {
		t.Fatalf("expected status %d, got %d; body=%s", want, resp.Code, resp.Body.String())
	}
}

func decodeEnvelope(t *testing.T, resp *httptest.ResponseRecorder) Envelope {
	t.Helper()
	var env Envelope
	if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode envelope: %v", err)
	}
	return env
}

func mustDataObject(t *testing.T, env Envelope) map[string]any {
	t.Helper()
	data, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", env.Data)
	}
	return data
}
