// Simplest possible tool — one input, one output.

package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func registerEcho(r *tools.Registry) {
	r.Register(tools.ToolDef{
		Name:        "echo",
		Description: "Echoes the input message back. Useful for testing connectivity.",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"message": {
					Type:        "string",
					Description: "The text to echo back",
				},
			},
			Required: []string{"message"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var params struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		return map[string]string{"echo": params.Message}, nil
	})
}
