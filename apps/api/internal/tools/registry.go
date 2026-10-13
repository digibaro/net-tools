// Package tools defines the contract every server-side tool implements and how its availability
// is reported through /api/v1/capabilities (docs/ARCHITECTURE.md §4).
package tools

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/config"
)

// Reason explains why a tool is disabled (docs/API_SPEC.md §2).
type Reason string

const (
	ReasonDisabledByAdmin    Reason = "disabled_by_admin"
	ReasonProviderKeyMissing Reason = "provider_key_missing"
	ReasonRequiresLANMode    Reason = "requires_lan_mode"
	ReasonBinaryMissing      Reason = "binary_missing"
)

// Deps are the shared dependencies handed to tools when they mount their routes.
// Phase 2 adds the target resolver, exec runner, rate limiter and concurrency pool.
type Deps struct {
	Config config.Config
	Logger *slog.Logger
}

// Tool is one server-side capability.
type Tool interface {
	// ID is the capability id, e.g. "dns.lookup".
	ID() string
	// Mount registers the tool's routes on a router rooted at /api/v1.
	Mount(r chi.Router, deps Deps)
	// Enabled reports whether the tool may run under cfg, and why not.
	Enabled(cfg config.Config) (bool, Reason)
}

// Capability is the per-tool entry of /api/v1/capabilities.
type Capability struct {
	Enabled bool   `json:"enabled"`
	Reason  Reason `json:"reason,omitempty"`
}

// Capabilities evaluates every tool against cfg.
func Capabilities(cfg config.Config, all []Tool) map[string]Capability {
	out := make(map[string]Capability, len(all))
	for _, t := range all {
		ok, reason := t.Enabled(cfg)
		c := Capability{Enabled: ok}
		if !ok {
			c.Reason = reason
		}
		out[t.ID()] = c
	}
	return out
}

// All returns the registered tools. Empty until Phase 2.
func All() []Tool {
	return []Tool{}
}
