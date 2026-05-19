package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type NotFoundError struct{ Name string }

func (e *NotFoundError) Error() string { return fmt.Sprintf("tool %q not found", e.Name) }

type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"input_schema"`
}

type ToolHandler func(ctx context.Context, input json.RawMessage) (any, error)

type registeredTool struct {
	def     ToolDef
	handler ToolHandler
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]registeredTool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]registeredTool)}
}

func (r *Registry) Register(def ToolDef, handler ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[def.Name] = registeredTool{def: def, handler: handler}
}

func (r *Registry) List() []ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.def)
	}
	return out
}

func (r *Registry) Invoke(ctx context.Context, name string, input json.RawMessage) (any, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return t.handler(ctx, input)
}
