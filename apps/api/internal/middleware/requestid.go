// Package middleware contains HTTP middleware shared by every route.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
)

// RequestIDHeader is echoed on every response.
const RequestIDHeader = "X-Request-Id"

type ctxKey int

const (
	requestIDKey ctxKey = iota
	loggerKey
	clientIPKey
)

// Accept upstream ids only if they look harmless, to keep logs clean.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// RequestID assigns an id (reusing a sane incoming X-Request-Id), sets the response header and
// stores a request-scoped logger carrying the id.
func RequestID(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if !validRequestID.MatchString(id) {
				id = newID()
			}
			w.Header().Set(RequestIDHeader, id)
			ctx := context.WithValue(r.Context(), requestIDKey, id)
			ctx = context.WithValue(ctx, loggerKey, base.With(slog.String("requestId", id)))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestID returns the request id, or "".
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// Logger returns the request-scoped logger, or slog.Default().
func Logger(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

func newID() string {
	var b [12]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read never returns an error
	return hex.EncodeToString(b[:])
}
