package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type NotFoundError struct{ Name string }

func (e *NotFoundError) Error() string { return fmt.Sprintf("tool %q not found", e.Name) }

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

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
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (r *Registry) Lookup(name string) (ToolDef, ToolHandler, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return ToolDef{}, nil, &NotFoundError{Name: name}
	}
	return t.def, t.handler, nil
}

func (r *Registry) Validate(name string, input json.RawMessage) error {
	def, _, err := r.Lookup(name)
	if err != nil {
		return err
	}
	return ValidateInput(def.InputSchema, input)
}

func (r *Registry) Invoke(ctx context.Context, name string, input json.RawMessage) (any, error) {
	_, handler, err := r.Lookup(name)
	if err != nil {
		return nil, err
	}
	if err := ValidateInputFromName(r, name, input); err != nil {
		return nil, err
	}
	return handler(ctx, input)
}

func ValidateInputFromName(r *Registry, name string, input json.RawMessage) error {
	return r.Validate(name, input)
}

func ValidateInput(schema InputSchema, input json.RawMessage) error {
	if len(bytesTrimSpace(input)) == 0 {
		input = json.RawMessage("{}")
	}

	var payload any
	if err := json.Unmarshal(input, &payload); err != nil {
		return &ValidationError{Message: fmt.Sprintf("invalid JSON input: %v", err)}
	}

	objectValue, ok := payload.(map[string]any)
	if !ok {
		return &ValidationError{Message: "input must be a JSON object"}
	}

	for _, field := range schema.Required {
		value, exists := objectValue[field]
		if !exists || value == nil {
			return &ValidationError{Message: fmt.Sprintf("field %q is required", field)}
		}
	}

	for _, name := range sortedObjectKeys(objectValue) {
		value := objectValue[name]
		property, exists := schema.Properties[name]
		if !exists {
			return &ValidationError{Message: fmt.Sprintf("field %q is not allowed", name)}
		}
		if err := validateProperty(name, property, value); err != nil {
			return err
		}
	}

	return nil
}

func validateProperty(name string, property PropertySchema, value any) error {
	if value == nil {
		return nil
	}

	switch property.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return &ValidationError{Message: fmt.Sprintf("field %q must be a string", name)}
		}
	case "integer":
		if !isJSONInteger(value) {
			return &ValidationError{Message: fmt.Sprintf("field %q must be an integer", name)}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return &ValidationError{Message: fmt.Sprintf("field %q must be a boolean", name)}
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return &ValidationError{Message: fmt.Sprintf("field %q must be an object", name)}
		}
	}

	if len(property.Enum) > 0 && !enumContains(property.Enum, value) {
		return &ValidationError{Message: fmt.Sprintf("field %q must be one of %s", name, joinEnumValues(property.Enum))}
	}

	return nil
}

func isJSONInteger(value any) bool {
	number, ok := value.(float64)
	if !ok {
		return false
	}
	return number == float64(int64(number))
}

func enumContains(options []any, value any) bool {
	for _, option := range options {
		if fmt.Sprint(option) == fmt.Sprint(value) {
			return true
		}
	}
	return false
}

func joinEnumValues(options []any) string {
	parts := make([]string, 0, len(options))
	for _, option := range options {
		parts = append(parts, fmt.Sprintf("%q", fmt.Sprint(option)))
	}
	return strings.Join(parts, ", ")
}

func bytesTrimSpace(input []byte) []byte {
	return []byte(strings.TrimSpace(string(input)))
}

func sortedObjectKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
