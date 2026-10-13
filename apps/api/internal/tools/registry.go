// Package tools defines the contract every server-side tool implements and how its availability
// is reported through /api/v1/capabilities (docs/ARCHITECTURE.md §4).
package tools

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/config"
	"github.com/digibaro/net-tools/apps/api/internal/execx"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
)

// Reason explains why a tool is disabled (docs/API_SPEC.md §2).
type Reason string

const (
	ReasonDisabledByAdmin    Reason = "disabled_by_admin"
	ReasonProviderKeyMissing Reason = "provider_key_missing"
	ReasonRequiresLANMode    Reason = "requires_lan_mode"
	ReasonBinaryMissing      Reason = "binary_missing"
	ReasonDataMissing        Reason = "data_missing"
)

// Middleware is a standard net/http middleware.
type Middleware = func(http.Handler) http.Handler

// Deps are the shared dependencies handed to tools.
type Deps struct {
	Config config.Config
	Logger *slog.Logger
	// Target is the only way to obtain an address to dial.
	Target *target.Resolver
	Exec   *execx.Runner
	Pool   *limits.Pool
	// RateLimit returns the per-client limiter middleware for a class.
	RateLimit func(limits.Class) Middleware
	// Timeout bounds a route's context.
	Timeout func(time.Duration) Middleware
	// Slot holds a concurrency-pool slot for the duration of the request (BUSY when full).
	Slot Middleware
	// UserAgent identifies outbound requests: NetTools/<version> (+PUBLIC_URL).
	UserAgent string
}

// Tool is one server-side capability.
type Tool interface {
	// ID is the capability id, e.g. "dns.lookup".
	ID() string
	// Mount registers the tool's routes on a router rooted at /api/v1.
	Mount(r chi.Router, deps Deps)
	// Enabled reports whether the tool may run, and why not.
	Enabled(deps Deps) (bool, Reason)
}

// Capability is the per-tool entry of /api/v1/capabilities.
type Capability struct {
	Enabled bool   `json:"enabled"`
	Reason  Reason `json:"reason,omitempty"`
}

// Capabilities evaluates every tool.
func Capabilities(d Deps, all []Tool) map[string]Capability {
	out := make(map[string]Capability, len(all))
	for _, t := range all {
		ok, reason := t.Enabled(d)
		c := Capability{Enabled: ok}
		if !ok {
			c.Reason = reason
		}
		out[t.ID()] = c
	}
	return out
}

// Always is an Enabled implementation for tools without preconditions.
func Always(Deps) (bool, Reason) { return true, "" }
