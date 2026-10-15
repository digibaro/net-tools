// Package stream runs a tool as an SSE stream: meta → typed events → done | error.
package stream

import (
	"context"
	"net/http"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/sse"
)

// Emit sends one typed event.
type Emit func(event string, data any)

// Job is prepared after validation; Run streams events and returns the done summary.
type Job struct {
	Meta any
	Run  func(ctx context.Context, emit Emit) (summary any, err error)
}

// ErrorData is the payload of the error event (docs/API_SPEC.md §4).
type ErrorData struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// DoneData is the payload of the done event.
type DoneData struct {
	Summary    any     `json:"summary"`
	DurationMs float64 `json:"durationMs"`
}

func errorData(err error) ErrorData {
	a := httpapi.ToAPIError(err)
	return ErrorData{Code: string(a.Code), Message: a.Message, Details: a.Details}
}

// Serve validates via prepare and streams the job. Every failure, including validation and
// target-policy errors, is delivered as an SSE "error" event so EventSource clients can read it.
func Serve(w http.ResponseWriter, r *http.Request, prepare func(ctx context.Context) (*Job, error)) {
	s := sse.New(w)
	defer s.Close()
	ctx := r.Context()
	job, err := prepare(ctx)
	if err != nil {
		_ = s.Event("error", errorData(err))
		return
	}
	if job.Meta != nil {
		_ = s.Event("meta", job.Meta)
	}
	start := time.Now()
	summary, err := job.Run(ctx, func(event string, data any) { _ = s.Event(event, data) })
	if err != nil {
		if ctx.Err() != nil && r.Context().Err() == context.Canceled {
			return // client went away
		}
		_ = s.Event("error", errorData(err))
		return
	}
	_ = s.Event("done", DoneData{Summary: summary, DurationMs: float64(time.Since(start).Microseconds()) / 1000})
}
