// Enum input — shows how to use the Enum field in PropertySchema and validate it.

package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func registerBase64(r *tools.Registry) {
	r.Register(tools.ToolDef{
		Name:        "base64",
		Description: "Encodes or decodes a string using Base64.",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"operation": {
					Type:        "string",
					Description: "Whether to encode or decode the input",
					Enum:        []any{"encode", "decode"},
				},
				"value": {
					Type:        "string",
					Description: "The string to encode or decode",
				},
			},
			Required: []string{"operation", "value"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var params struct {
			Operation string `json:"operation"`
			Value     string `json:"value"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}

		switch params.Operation {
		case "encode":
			return map[string]string{
				"result": base64.StdEncoding.EncodeToString([]byte(params.Value)),
			}, nil
		case "decode":
			decoded, err := base64.StdEncoding.DecodeString(params.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid base64 string: %w", err)
			}
			return map[string]string{"result": string(decoded)}, nil
		default:
			return nil, fmt.Errorf("operation must be 'encode' or 'decode', got %q", params.Operation)
		}
	})
}
