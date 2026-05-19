package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func TestFileTools(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	registry := tools.NewRegistry()
	config := Config{WorkspaceRoot: root}
	registerFileList(registry, config)
	registerFileRead(registry, config)
	registerFileWrite(registry, config)

	tests := []struct {
		name        string
		tool        string
		input       string
		wantErrPart string
		assert      func(t *testing.T, result any)
	}{
		{
			name:  "file_write creates nested file",
			tool:  "file_write",
			input: `{"path":"notes/a.txt","content":"hello","create_directories":true}`,
			assert: func(t *testing.T, result any) {
				t.Helper()
				payload := mustMap(t, result)
				if payload["path"] != "notes/a.txt" {
					t.Fatalf("unexpected path: %#v", payload["path"])
				}
				if payload["mode"] != "overwrite" {
					t.Fatalf("unexpected mode: %#v", payload["mode"])
				}
			},
		},
		{
			name:  "file_read returns content",
			tool:  "file_read",
			input: `{"path":"notes/a.txt"}`,
			assert: func(t *testing.T, result any) {
				t.Helper()
				payload := mustMap(t, result)
				if payload["content"] != "hello" {
					t.Fatalf("unexpected content: %#v", payload["content"])
				}
			},
		},
		{
			name:  "file_write append mode",
			tool:  "file_write",
			input: `{"path":"notes/a.txt","content":" world","mode":"append"}`,
			assert: func(t *testing.T, result any) {
				t.Helper()
				data, err := os.ReadFile(filepath.Join(root, "notes", "a.txt"))
				if err != nil {
					t.Fatalf("failed to read file after append: %v", err)
				}
				if string(data) != "hello world" {
					t.Fatalf("unexpected appended content: %q", string(data))
				}
			},
		},
		{
			name:  "file_list root includes directory",
			tool:  "file_list",
			input: `{"path":"."}`,
			assert: func(t *testing.T, result any) {
				t.Helper()
				payload := mustMap(t, result)
				entries := payload["entries"]
				v := reflect.ValueOf(entries)
				if v.Kind() != reflect.Slice {
					t.Fatalf("entries not slice: %T", entries)
				}
				if v.Len() == 0 {
					t.Fatalf("expected at least one entry")
				}
			},
		},
		{
			name:        "file_read blocks path traversal",
			tool:        "file_read",
			input:       `{"path":"../outside.txt"}`,
			wantErrPart: "escapes the workspace root",
		},
		{
			name:        "file_write create_new fails when exists",
			tool:        "file_write",
			input:       `{"path":"notes/a.txt","content":"again","mode":"create_new"}`,
			wantErrPart: "failed to open file for write",
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

func mustMap(t *testing.T, value any) map[string]any {
	t.Helper()
	out, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", value)
	}
	return out
}
