package builtin

import (
	"github.com/seanrobmerriam/go-agent-rest-api/internal/tools"
)

type Config struct {
	WorkspaceRoot string
}

// Register adds all built-in tools to the registry.
// Call this from main.go before starting the server.
func Register(r *tools.Registry, cfg Config) {
	registerFileList(r, cfg)
	registerFileRead(r, cfg)
	registerFileWrite(r, cfg)
	registerWordCount(r)
	registerBase64(r)
	registerHTTPGet(r)
	registerHTTPRequest(r)
	registerJSONValidate(r)
}
