package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func registerFileList(r *tools.Registry, cfg Config) {
	r.Register(tools.ToolDef{
		Name:        "file_list",
		Description: "Lists files and directories under the configured workspace root.",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"path": {
					Type:        "string",
					Description: "Relative directory path inside the workspace root. Use '.' for the root.",
				},
			},
			Required: []string{"path"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var params struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}

		targetPath, relativePath, err := resolveWorkspacePath(cfg.WorkspaceRoot, params.Path)
		if err != nil {
			return nil, err
		}

		entries, err := os.ReadDir(targetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to list directory: %w", err)
		}

		type entry struct {
			Name  string `json:"name"`
			Path  string `json:"path"`
			IsDir bool   `json:"is_dir"`
		}

		out := make([]entry, 0, len(entries))
		for _, item := range entries {
			itemPath := item.Name()
			if relativePath != "." {
				itemPath = filepath.ToSlash(filepath.Join(relativePath, item.Name()))
			}
			out = append(out, entry{
				Name:  item.Name(),
				Path:  filepath.ToSlash(itemPath),
				IsDir: item.IsDir(),
			})
		}

		sort.Slice(out, func(i, j int) bool {
			if out[i].IsDir != out[j].IsDir {
				return out[i].IsDir
			}
			return out[i].Path < out[j].Path
		})

		return map[string]any{
			"path":    filepath.ToSlash(relativePath),
			"entries": out,
		}, nil
	})
}

func registerFileRead(r *tools.Registry, cfg Config) {
	r.Register(tools.ToolDef{
		Name:        "file_read",
		Description: "Reads a UTF-8 text file from the configured workspace root. Output is capped at 16384 bytes.",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"path": {
					Type:        "string",
					Description: "Relative file path inside the workspace root.",
				},
			},
			Required: []string{"path"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var params struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}

		targetPath, relativePath, err := resolveWorkspacePath(cfg.WorkspaceRoot, params.Path)
		if err != nil {
			return nil, err
		}

		file, err := os.Open(targetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to stat file: %w", err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("path %q is a directory", relativePath)
		}

		const maxBytes = 16 * 1024
		content, err := io.ReadAll(io.LimitReader(file, maxBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		return map[string]any{
			"path":      filepath.ToSlash(relativePath),
			"content":   string(content),
			"truncated": info.Size() > int64(len(content)),
			"size":      info.Size(),
		}, nil
	})
}

func registerFileWrite(r *tools.Registry, cfg Config) {
	r.Register(tools.ToolDef{
		Name:        "file_write",
		Description: "Writes UTF-8 text to a file under the configured workspace root. Supports overwrite, append, and create-only modes.",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"path": {
					Type:        "string",
					Description: "Relative file path inside the workspace root.",
				},
				"content": {
					Type:        "string",
					Description: "UTF-8 text to write to the file.",
				},
				"mode": {
					Type:        "string",
					Description: "Write mode: overwrite, append, or create_new. Defaults to overwrite.",
					Enum:        []any{"overwrite", "append", "create_new"},
				},
				"create_directories": {
					Type:        "boolean",
					Description: "Create parent directories when they do not exist.",
				},
			},
			Required: []string{"path", "content"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var params struct {
			Path              string `json:"path"`
			Content           string `json:"content"`
			Mode              string `json:"mode"`
			CreateDirectories bool   `json:"create_directories"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}

		targetPath, relativePath, err := resolveWorkspacePath(cfg.WorkspaceRoot, params.Path)
		if err != nil {
			return nil, err
		}

		mode := params.Mode
		if mode == "" {
			mode = "overwrite"
		}

		if params.CreateDirectories {
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return nil, fmt.Errorf("failed to create parent directories: %w", err)
			}
		}

		flags := os.O_WRONLY
		switch mode {
		case "overwrite":
			flags |= os.O_CREATE | os.O_TRUNC
		case "append":
			flags |= os.O_CREATE | os.O_APPEND
		case "create_new":
			flags |= os.O_CREATE | os.O_EXCL
		default:
			return nil, fmt.Errorf("mode must be one of overwrite, append, create_new")
		}

		file, err := os.OpenFile(targetPath, flags, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for write: %w", err)
		}
		defer file.Close()

		written, err := file.WriteString(params.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}

		info, err := file.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to stat written file: %w", err)
		}

		return map[string]any{
			"path":          filepath.ToSlash(relativePath),
			"bytes_written": written,
			"mode":          mode,
			"size":          info.Size(),
		}, nil
	})
}

func resolveWorkspacePath(root, requestedPath string) (string, string, error) {
	if root == "" {
		return "", "", fmt.Errorf("workspace root is not configured")
	}

	requestedPath = strings.TrimSpace(requestedPath)
	if requestedPath == "" {
		requestedPath = "."
	}
	if filepath.IsAbs(requestedPath) {
		return "", "", fmt.Errorf("path must be relative to the workspace root")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve workspace root: %w", err)
	}

	targetPath := filepath.Join(absRoot, requestedPath)
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve path: %w", err)
	}

	relToRoot, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", "", fmt.Errorf("failed to compare path against workspace root: %w", err)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path %q escapes the workspace root", requestedPath)
	}

	cleanRelative := filepath.Clean(relToRoot)
	if cleanRelative == "." {
		return absTarget, cleanRelative, nil
	}

	return absTarget, cleanRelative, nil
}
