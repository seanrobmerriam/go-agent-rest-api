// schema.go defines JSON schemas for API request and response validation.
package tools

// InputSchema is a JSON Schema "object" descriptor for a tool's accepted input.
type InputSchema struct {
	Type       string                    `json:"type"` // always "object"
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

// PropertySchema describes a single field within an InputSchema.
type PropertySchema struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Enum        []any  `json:"enum,omitempty"`
}
