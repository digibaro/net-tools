package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/nettools/apps/api/internal/config"
	"github.com/digibaro/nettools/apps/api/internal/middleware"
	"github.com/digibaro/nettools/apps/api/internal/tools"
)

// maxRequestBody caps request bodies for every route; tools may lower it.
const maxRequestBody = 1 << 20

// BuildInfo is injected at link time (see docker/api.Dockerfile).
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
}

// ReadyCheck reports whether a dependency is ready; name appears in /api/readyz output.
type ReadyCheck struct {
	Name  string
	Check func(context.Context) error
}

// Options configures the router.
type Options struct {
	Config config.Config
	Logger *slog.Logger
	Build  BuildInfo
	Tools  []tools.Tool
	Ready  []ReadyCheck
}

// NewRouter builds the API handler. All routes live under /api.
func NewRouter(o Options) http.Handler {
	r := chi.NewRouter()
	r.Use(
		middleware.RequestID(o.Logger),
		middleware.ClientIP(o.Config.Features.TrustedProxies),
		middleware.AccessLog(ipFormatter(o.Config.Features.LogHashIPs)),
		middleware.Recover(func(w http.ResponseWriter, r *http.Request) {
			Error(w, r, NewError(CodeInternal, "internal error"))
		}),
		limitBody(maxRequestBody),
	)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		Error(w, r, NewError(CodeNotFound, "no such endpoint"))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		Error(w, r, NewError(CodeMethodNotAllowed, "method not allowed"))
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/healthz", handleHealthz)
		r.Get("/readyz", handleReadyz(o.Ready))

		r.Route("/v1", func(r chi.Router) {
			r.Get("/version", handleVersion(o.Build))
			r.Get("/capabilities", handleCapabilities(o.Config, o.Tools))
			r.Get("/echo", handleEcho(o.Config.Features.TrustedProxies))

			deps := tools.Deps{Config: o.Config, Logger: o.Logger}
			for _, t := range o.Tools {
				if ok, _ := t.Enabled(o.Config); ok {
					t.Mount(r, deps)
				}
			}
		})
	})
	return r
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func handleReadyz(checks []ReadyCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		failed := map[string]string{}
		for _, c := range checks {
			if err := c.Check(ctx); err != nil {
				middleware.Logger(ctx).WarnContext(ctx, "readiness check failed",
					slog.String("check", c.Name), slog.Any("err", err))
				failed[c.Name] = "not ready"
			}
		}
		if len(failed) > 0 {
			writeJSON(w, r, http.StatusServiceUnavailable, map[string]any{"status": "not_ready", "checks": failed})
			return
		}
		writeJSON(w, r, http.StatusOK, map[string]string{"status": "ready"})
	}
}

func handleVersion(b BuildInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		JSON(w, r, b, nil)
	}
}

// CapabilitiesData is the payload of /api/v1/capabilities.
type CapabilitiesData struct {
	Mode   ModeInfo                    `json:"mode"`
	Tools  map[string]tools.Capability `json:"tools"`
	Limits LimitsInfo                  `json:"limits"`
}

// ModeInfo describes the deployment mode.
type ModeInfo struct {
	Profile             config.Profile `json:"profile"`
	AllowPrivateTargets bool           `json:"allowPrivateTargets"`
	LAN                 bool           `json:"lan"`
}

// LimitsInfo exposes the limits the UI needs to bound its inputs.
type LimitsInfo struct {
	PingMaxCount      int   `json:"pingMaxCount"`
	PingMinIntervalMs int64 `json:"pingMinIntervalMs"`
	TracerouteMaxHops int   `json:"tracerouteMaxHops"`
	MTRMaxCycles      int   `json:"mtrMaxCycles"`
	PortscanMaxPorts  int   `json:"portscanMaxPorts"`
	IperfMaxSeconds   int   `json:"iperfMaxSeconds"`
	BulkMaxTargets    int   `json:"bulkMaxTargets"`
}

func handleCapabilities(cfg config.Config, all []tools.Tool) http.HandlerFunc {
	data := CapabilitiesData{
		Mode: ModeInfo{
			Profile:             cfg.Profile,
			AllowPrivateTargets: cfg.Features.AllowPrivate,
			LAN:                 cfg.Features.LAN,
		},
		Tools: tools.Capabilities(cfg, all),
		Limits: LimitsInfo{
			PingMaxCount:      cfg.Limits.PingMaxCount,
			PingMinIntervalMs: cfg.Limits.PingMinInterval.Milliseconds(),
			TracerouteMaxHops: cfg.Limits.TracerouteMaxHops,
			MTRMaxCycles:      cfg.Limits.MTRMaxCycles,
			PortscanMaxPorts:  cfg.Limits.PortscanMaxPorts,
			IperfMaxSeconds:   cfg.Limits.IperfMaxSeconds,
			BulkMaxTargets:    cfg.Limits.BulkMaxTargets,
		},
	}
	return func(w http.ResponseWriter, r *http.Request) {
		JSON(w, r, data, nil)
	}
}

func limitBody(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ipFormatter(hash bool) func(string) string {
	if !hash {
		return func(s string) string { return s }
	}
	return func(s string) string {
		sum := sha256.Sum256([]byte(s))
		return hex.EncodeToString(sum[:8])
	}
}
