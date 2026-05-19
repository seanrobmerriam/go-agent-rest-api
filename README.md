# go-agent-rest-api

A small Go REST API for agent-friendly tool discovery and invocation.

The server exposes a registry of tools, supports direct tool execution, and provides async job execution endpoints for long-running work.

## Features

- Tool discovery endpoint with JSON Schema-style input definitions
- Strict input validation before dispatch (required fields, types, enums, and unknown field rejection)
- Synchronous tool invocation endpoint
- Async jobs API (`POST /v1/jobs` + `GET /v1/jobs/{id}`)
- Consistent JSON response envelope for success and error responses
- Optional API key authentication via `API_KEY`
- Built-in tools:
  - `file_list`
  - `file_read`
  - `file_write`
  - `word_count`
  - `base64`
  - `http_get`
  - `http_request`
  - `json_validate`

## Requirements

- Go 1.24+

## Run Locally

```bash
go mod tidy
go run ./cmd/server
```

By default, the server listens on `:8080`.

You can override configuration with environment variables:

```bash
export ADDR=":8080"
export API_KEY="your-api-key"  # optional; if empty, auth is disabled
export WORKSPACE_ROOT="$(pwd)"  # optional; defaults to the current working directory
go run ./cmd/server
```

## API Endpoints

- `GET /v1/health`
  - Liveness endpoint, no auth required.
- `GET /v1/tools`
  - Returns registered tools and their input schemas.
- `POST /v1/tools/{name}`
  - Invokes a tool synchronously.
- `POST /v1/jobs`
  - Starts a background job to run a tool.
- `GET /v1/jobs/{id}`
  - Fetches job status/result.

If `API_KEY` is set, all endpoints except `GET /v1/health` require:

```http
Authorization: Bearer <API_KEY>
```

Auth rules:

- The `Authorization` header must use the `Bearer ` scheme exactly. A raw key without the prefix is rejected.
- An empty or missing header is rejected with `401 UNAUTHORIZED`.
- If `API_KEY` is not set (dev mode), auth is skipped for all endpoints.

## Validation Rules

Tool input is validated against each tool's registered `InputSchema` before handler execution.

- Input must be a JSON object.
- Required fields must be present.
- Field types are enforced (`string`, `integer`, `boolean`, `object`).
- Enum-constrained fields must match one of the allowed values.
- Unknown fields are rejected.

Validation failures return `400` with `error.code = INVALID_INPUT`.

## Response Envelope

Success:

```json
{
  "ok": true,
  "data": { "...": "..." },
  "error": null
}
```

Error:

```json
{
  "ok": false,
  "data": null,
  "error": {
    "code": "INVALID_INPUT",
    "message": "field 'tool' is required"
  }
}
```

Unknown field example:

```json
{
  "ok": false,
  "data": null,
  "error": {
    "code": "INVALID_INPUT",
    "message": "field \"extra\" is not allowed"
  }
}
```

## Quick Examples

Health check:

```bash
curl -s http://localhost:8080/v1/health
```

List tools (no API key configured):

```bash
curl -s http://localhost:8080/v1/tools | jq
```

Invoke `file_list`:

```bash
curl -s -X POST http://localhost:8080/v1/tools/file_list \
  -H 'Content-Type: application/json' \
  -d '{"path":"."}' | jq
```

Invoke `file_read`:

```bash
curl -s -X POST http://localhost:8080/v1/tools/file_read \
  -H 'Content-Type: application/json' \
  -d '{"path":"README.md"}' | jq
```

Invoke `file_write`:

```bash
curl -s -X POST http://localhost:8080/v1/tools/file_write \
  -H 'Content-Type: application/json' \
  -d '{"path":"tmp/example.txt","content":"hello\n","mode":"overwrite","create_directories":true}' | jq
```

Invoke `http_request`:

```bash
curl -s -X POST http://localhost:8080/v1/tools/http_request \
  -H 'Content-Type: application/json' \
  -d '{"method":"GET","url":"https://example.com","headers":{"Accept":"text/html"}}' | jq
```

Create async job:

```bash
curl -s -X POST http://localhost:8080/v1/jobs \
  -H 'Content-Type: application/json' \
  -d '{"tool":"word_count","input":{"text":"hello world"}}' | jq
```

Then poll using returned `id`:

```bash
curl -s http://localhost:8080/v1/jobs/<id> | jq
```

## Testing

Run the full suite:

```bash
go test ./...
```

Current coverage includes:

- router behavior (`/v1/health`, `/v1/tools/{name}`, `/v1/jobs`, `/v1/jobs/{id}`)
- auth middleware: dev-mode bypass, correct token, wrong token, missing header, raw key without `Bearer` prefix, alternate scheme
- schema validation: required fields, type checks, enum checks, unknown field rejection, malformed JSON
- built-in file tools: write, append, read, list, path traversal rejection, create-new conflict
- built-in HTTP tools: GET success, POST with headers and body, invalid URL

## Project Layout

```text
cmd/server/main.go           # Server bootstrap and env config
internal/api/                # Router, middleware, response helpers
internal/tools/              # Tool registry and schema definitions
internal/tools/builtin/      # Built-in tool implementations
internal/jobs/store.go       # In-memory async job store
```
