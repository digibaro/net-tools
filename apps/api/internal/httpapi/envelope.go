// Package httpapi holds the HTTP router, the JSON envelope and error mapping shared by all tools.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/digibaro/net-tools-docker/apps/api/internal/middleware"
)

// Meta is the optional envelope metadata (docs/API_SPEC.md §1.1).
type Meta struct {
	Tool       string      `json:"tool,omitempty"`
	Target     *TargetMeta `json:"target,omitempty"`
	Vantage    string      `json:"vantage,omitempty"`
	DurationMs float64     `json:"durationMs,omitempty"`
	Cached     bool        `json:"cached,omitempty"`
}

// TargetMeta describes how a user-supplied target was resolved.
type TargetMeta struct {
	Input    string   `json:"input"`
	Resolved []string `json:"resolved,omitempty"`
	Used     string   `json:"used,omitempty"`
}

type okEnvelope struct {
	OK   bool  `json:"ok"`
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

type errEnvelope struct {
	OK    bool      `json:"ok"`
	Error *APIError `json:"error"`
}

// JSON writes a success envelope.
func JSON(w http.ResponseWriter, r *http.Request, data any, meta *Meta) {
	writeJSON(w, r, http.StatusOK, okEnvelope{OK: true, Data: data, Meta: meta})
}

// Error writes an error envelope. Internal errors are logged with the request id.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := ToAPIError(err)
	status := apiErr.Code.Status()
	if status >= http.StatusInternalServerError {
		middleware.Logger(r.Context()).ErrorContext(r.Context(), "request failed",
			slog.String("code", string(apiErr.Code)), slog.Any("err", err))
	}
	writeJSON(w, r, status, errEnvelope{OK: false, Error: apiErr})
}

// Respond writes data or, if err is non-nil, the error envelope.
func Respond(w http.ResponseWriter, r *http.Request, data any, meta *Meta, err error) {
	if err != nil {
		Error(w, r, err)
		return
	}
	JSON(w, r, data, meta)
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		middleware.Logger(r.Context()).WarnContext(r.Context(), "write response", slog.Any("err", err))
	}
}
