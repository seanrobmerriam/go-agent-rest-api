// Error-as-data pattern — shows returning a structured result even for failure cases rather than propagating an error.

package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func registerJSONValidate(r *tools.Registry) {
	r.Register(tools.ToolDef{
		Name:        "json_validate",
		Description: "Checks whether a string is valid JSON. Returns valid:true/false and an error message if invalid. Does not return a tool error — the result itself carries the validation outcome.",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"value": {
					Type:        "string",
					Description: "The string to validate as JSON",
				},
			},
			Required: []string{"value"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var params struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}

		var probe any
		if err := json.Unmarshal([]byte(params.Value), &probe); err != nil {
			// Not a tool error — validation failure is a normal result
			return map[string]any{
				"valid":   false,
				"message": err.Error(),
			}, nil
		}

		return map[string]any{
			"valid":   true,
			"message": "valid JSON",
		}, nil
	})
}