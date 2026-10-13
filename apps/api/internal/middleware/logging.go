package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder captures the status code; it forwards Flush so SSE keeps working.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err //nolint:wrapcheck // transparent ResponseWriter wrapper
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// AccessLog logs one line per request. ipFormatter controls how the client IP appears
// (e.g. hashed when LOG_HASH_IPS=true).
func AccessLog(ipFormatter func(string) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			if rec.status == 0 {
				rec.status = http.StatusOK
			}
			level := slog.LevelInfo
			if rec.status >= http.StatusInternalServerError {
				level = slog.LevelError
			}
			client := ""
			if ip := GetClientIP(r.Context()); ip.IsValid() {
				client = ipFormatter(ip.String())
			}
			Logger(r.Context()).LogAttrs(r.Context(), level, "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Int("bytes", rec.bytes),
				slog.String("clientIp", client),
				slog.Float64("durationMs", float64(time.Since(start).Microseconds())/1000),
			)
		})
	}
}

// Recover turns panics into a 500 and logs the stack. onPanic writes the response.
func Recover(onPanic func(http.ResponseWriter, *http.Request)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer recoverPanic(w, r, onPanic)
			next.ServeHTTP(w, r)
		})
	}
}

func recoverPanic(w http.ResponseWriter, r *http.Request, onPanic func(http.ResponseWriter, *http.Request)) {
	rec := recover()
	if rec == nil {
		return
	}
	if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
		panic(rec)
	}
	Logger(r.Context()).ErrorContext(r.Context(), "panic",
		slog.Any("panic", rec), slog.String("stack", string(debug.Stack())))
	onPanic(w, r)
}
