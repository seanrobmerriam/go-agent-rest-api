package builtin

import (
	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

// Register adds all built-in example tools to the registry.
// Call this from main.go before starting the server.
func Register(r *tools.Registry) {
	registerEcho(r)
	registerWordCount(r)
	registerBase64(r)
	registerHTTPGet(r)
	registerJSONValidate(r)
}
