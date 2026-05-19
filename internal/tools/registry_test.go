package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValidateInput(t *testing.T) {
	t.Parallel()

	schema := InputSchema{
		Type: "object",
		Properties: map[string]PropertySchema{
			"name": {
				Type: "string",
			},
			"count": {
				Type: "integer",
			},
			"enabled": {
				Type: "boolean",
			},
			"mode": {
				Type: "string",
				Enum: []any{"fast", "safe"},
			},
			"metadata": {
				Type: "object",
			},
		},
		Required: []string{"name", "count"},
	}

	tests := []struct {
		name       string
		input      string
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid payload",
			input:   `{"name":"test","count":2,"enabled":true,"mode":"fast","metadata":{"k":"v"}}`,
			wantErr: false,
		},
		{
			name:       "missing required field",
			input:      `{"name":"test"}`,
			wantErr:    true,
			errContain: `field "count" is required`,
		},
		{
			name:       "wrong type",
			input:      `{"name":"test","count":"2"}`,
			wantErr:    true,
			errContain: `field "count" must be an integer`,
		},
		{
			name:       "enum violation",
			input:      `{"name":"test","count":2,"mode":"turbo"}`,
			wantErr:    true,
			errContain: `field "mode" must be one of`,
		},
		{
			name:       "input is not object",
			input:      `[]`,
			wantErr:    true,
			errContain: "input must be a JSON object",
		},
		{
			name:       "unknown field rejected",
			input:      `{"name":"test","count":2,"extra":"x"}`,
			wantErr:    true,
			errContain: `field "extra" is not allowed`,
		},
		{
			name:       "invalid json",
			input:      `{"name":`,
			wantErr:    true,
			errContain: "invalid JSON input",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateInput(schema, json.RawMessage(tc.input))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.wantErr && tc.errContain != "" && !strings.Contains(err.Error(), tc.errContain) {
				t.Fatalf("expected error to contain %q, got %q", tc.errContain, err.Error())
			}
		})
	}
}

func TestRegistryInvokeValidatesBeforeHandler(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	handlerCalled := false
	r.Register(ToolDef{
		Name:        "sum",
		Description: "sum two integers",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"a": {Type: "integer"},
				"b": {Type: "integer"},
			},
			Required: []string{"a", "b"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		handlerCalled = true
		return map[string]int{"sum": 3}, nil
	})

	_, err := r.Invoke(context.Background(), "sum", json.RawMessage(`{"a":1}`))
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if handlerCalled {
		t.Fatalf("expected handler not to run when validation fails")
	}
}

func TestRegistryLookupNotFound(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	_, _, err := r.Lookup("missing")
	if err == nil {
		t.Fatalf("expected not found error")
	}

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("expected NotFoundError, got %T", err)
	}
}
