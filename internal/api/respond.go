package api

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data"`
	Error *APIError   `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Stable machine-readable error codes
const (
	ErrInvalidInput = "INVALID_INPUT"
	ErrNotFound     = "NOT_FOUND"
	ErrToolError    = "TOOL_ERROR"
	ErrUnauthorized = "UNAUTHORIZED"
	ErrInternal     = "INTERNAL_ERROR"
)

func Respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Envelope{OK: status < 400, Data: data})
}

func RespondErr(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Envelope{
		OK:    false,
		Error: &APIError{Code: code, Message: message},
	})
}
