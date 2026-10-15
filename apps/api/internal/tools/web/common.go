// Package web implements the HTTP & Web server tools (WEB-01, 02, 03, 04, 06).
package web

import (
	"context"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
)

const requestTimeout = 15 * time.Second

func fetch(ctx context.Context, d tools.Deps, req httpcore.Request) ([]httpcore.Hop, error) {
	req.UserAgent = d.UserAgent
	req.MaxBody = d.Config.Limits.HTTPMaxBodyBytes
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Timeout == 0 {
		req.Timeout = requestTimeout
	}
	return (&httpcore.Client{Resolver: d.Target}).Do(ctx, req) //nolint:wrapcheck // client-facing errors
}

func header(h httpcore.Hop, name string) (string, bool) {
	for _, x := range h.Headers {
		if equalFold(x.Name, name) {
			return x.Value, true
		}
	}
	return "", false
}

func headers(h httpcore.Hop, name string) []string {
	var out []string
	for _, x := range h.Headers {
		if equalFold(x.Name, name) {
			out = append(out, x.Value)
		}
	}
	return out
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
