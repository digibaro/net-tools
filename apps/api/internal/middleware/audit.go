package middleware

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// auditParams are the query parameters that name what a tool was pointed at, in priority order.
var auditParams = []string{"target", "host", "domain", "name", "url", "ip", "server", "set", "cidr", "prefix", "asn", "zone", "query", "endpoint", "targets", "mac"}

// AuditTarget extracts the target of a tool request for the audit log. URL values lose their
// query, fragment and credentials (they may carry tokens); everything is cut to 200 characters.
// Request bodies are never read, so POSTed secrets (SNMP communities, HTTP bodies) stay out.
func AuditTarget(q url.Values) string {
	for _, k := range auditParams {
		v := strings.TrimSpace(q.Get(k))
		if v == "" {
			continue
		}
		if u, err := url.Parse(v); err == nil && u.Scheme != "" && u.Host != "" {
			u.RawQuery, u.Fragment, u.User = "", "", nil
			v = u.String()
		}
		if len(v) > 200 {
			v = v[:200]
		}
		return v
	}
	return ""
}

// Audit logs one "audit" line per request to a tool when it finishes (streams included):
// which tool, what target, the outcome and the client (hashed with LOG_HASH_IPS).
func Audit(toolID string, ipFormatter func(string) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			if rec.status == 0 {
				rec.status = http.StatusOK
			}
			client := ""
			if ip := GetClientIP(r.Context()); ip.IsValid() {
				client = ipFormatter(ip.String())
			}
			Logger(r.Context()).LogAttrs(r.Context(), slog.LevelInfo, "audit",
				slog.String("tool", toolID),
				slog.String("method", r.Method),
				slog.String("target", AuditTarget(r.URL.Query())),
				slog.Int("status", rec.status),
				slog.String("clientIp", client),
				slog.Float64("durationMs", float64(time.Since(start).Microseconds())/1000),
			)
		})
	}
}
