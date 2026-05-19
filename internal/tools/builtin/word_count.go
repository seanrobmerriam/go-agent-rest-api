// Multiple output fields — shows that data can be any shape.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

func registerWordCount(r *tools.Registry) {
	r.Register(tools.ToolDef{
		Name:        "word_count",
		Description: "Counts the words, lines, and characters in a block of text.",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"text": {
					Type:        "string",
					Description: "The text to analyse",
				},
			},
			Required: []string{"text"},
		},
	}, func(ctx context.Context, input json.RawMessage) (any, error) {
		var params struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}

		words := 0
		if strings.TrimSpace(params.Text) != "" {
			words = len(strings.Fields(params.Text))
		}

		return map[string]int{
			"characters": len(params.Text),
			"words":      words,
			"lines":      len(strings.Split(params.Text, "\n")),
		}, nil
	})
}
