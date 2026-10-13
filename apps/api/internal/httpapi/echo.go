package httpapi

import (
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"github.com/digibaro/nettools/apps/api/internal/middleware"
)

// redactedHeaders may carry credentials (e.g. added by an auth proxy) and are never echoed.
var redactedHeaders = map[string]bool{
	"Authorization":       true,
	"Proxy-Authorization": true,
	"Cookie":              true,
}

// EchoData is the payload of /api/v1/echo.
type EchoData struct {
	IP        string       `json:"ip"`
	Family    int          `json:"family"`
	Protocol  string       `json:"protocol"`
	Scheme    string       `json:"scheme"`
	Method    string       `json:"method"`
	UserAgent string       `json:"userAgent"`
	Headers   []EchoHeader `json:"headers"`
}

// EchoHeader is one request header as received by the API.
type EchoHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func handleEcho(trusted []netip.Prefix) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := middleware.GetClientIP(r.Context())
		data := EchoData{
			Protocol:  r.Proto,
			Scheme:    scheme(r, trusted),
			Method:    r.Method,
			UserAgent: r.UserAgent(),
			Headers:   []EchoHeader{},
		}
		if ip.IsValid() {
			data.IP = ip.String()
			data.Family = 4
			if ip.Is6() {
				data.Family = 6
			}
		}
		for name, values := range r.Header {
			value := strings.Join(values, ", ")
			if redactedHeaders[name] {
				value = "[redacted]"
			}
			data.Headers = append(data.Headers, EchoHeader{Name: name, Value: value})
		}
		slices.SortFunc(data.Headers, func(a, b EchoHeader) int { return strings.Compare(a.Name, b.Name) })
		JSON(w, r, data, nil)
	}
}

func scheme(r *http.Request, trusted []netip.Prefix) string {
	if peer, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		for _, p := range trusted {
			if p.Contains(peer.Addr().Unmap()) {
				if proto := strings.ToLower(r.Header.Get("X-Forwarded-Proto")); proto == "http" || proto == "https" {
					return proto
				}
				break
			}
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
