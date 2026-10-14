// Package ripestat implements the RIPEstat-backed BGP tools: ASN info (BGP-02), prefix
// routing status (BGP-03), RPKI validation (BGP-04) and looking glass (BGP-05).
package ripestat

import (
	"context"
	"encoding/json"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/providers"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/whois"
)

// DefaultBase is the RIPEstat Data API.
const DefaultBase = "https://stat.ripe.net/data"

// Client calls RIPEstat data calls through the shared provider client.
type Client struct {
	// Base overrides DefaultBase (tests).
	Base string
}

type envelope struct {
	Status   string          `json:"status"`
	Messages [][]string      `json:"messages"`
	Data     json.RawMessage `json:"data"`
}

// Call runs data call `name` (e.g. "as-overview") and decodes `data` into out.
func (c Client) Call(ctx context.Context, d tools.Deps, name string, params url.Values, out any) error {
	base := c.Base
	if base == "" {
		base = DefaultBase
	}
	q := url.Values{}
	for k, v := range params {
		q[k] = v
	}
	q.Set("sourceapp", d.Config.Features.RIPEstatSrcApp) // RIPEstat asks clients to identify themselves
	var env envelope
	if err := d.Providers.GetJSON(ctx, providers.Request{Provider: "RIPEstat", URL: base + "/" + name + "/data.json?" + q.Encode(), UpstreamMessage: messageOf}, &env); err != nil {
		return err //nolint:wrapcheck // client-safe provider error
	}
	if env.Status != "ok" {
		return &providers.Error{Provider: "RIPEstat", Message: "RIPEstat: " + firstMessage(env.Messages, "the query failed")}
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return &providers.Error{Provider: "RIPEstat", Message: "RIPEstat returned an unexpected response", Err: err}
	}
	return nil
}

func messageOf(b []byte) string {
	var env envelope
	if json.Unmarshal(b, &env) == nil {
		return firstMessage(env.Messages, "")
	}
	return ""
}

func firstMessage(msgs [][]string, def string) string {
	for _, m := range msgs {
		if len(m) == 2 && (m[0] == "error" || m[0] == "warning") {
			return m[1]
		}
	}
	return def
}

// FlexASN decodes an ASN given as a number or a string ("13335" or "AS13335").
type FlexASN uint32

// UnmarshalJSON accepts both forms.
func (f *FlexASN) UnmarshalJSON(b []byte) error {
	s := strings.TrimPrefix(strings.ToUpper(strings.Trim(string(b), `"`)), "AS")
	if s == "" || s == "NULL" {
		*f = 0
		return nil
	}
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err //nolint:wrapcheck // decode error
	}
	*f = FlexASN(n)
	return nil
}

// ParseASN accepts "AS13335" or "13335".
func ParseASN(raw, field string) (uint32, error) {
	q, kind, err := whois.Normalize(raw)
	if err != nil || kind != "asn" {
		return 0, httpapi.InvalidInput(field, "enter an AS number such as AS13335")
	}
	n, _ := strconv.ParseUint(strings.TrimPrefix(q, "AS"), 10, 32)
	return uint32(n), nil
}

var (
	cgnat     = netip.MustParsePrefix("100.64.0.0/10")
	reserved4 = netip.MustParsePrefix("240.0.0.0/4")
)

// ParsePrefix accepts a prefix or a single address (a host route) and rejects space that is never
// routed on the Internet.
func ParsePrefix(raw, field string) (netip.Prefix, error) {
	s := strings.TrimSpace(raw)
	var p netip.Prefix
	if strings.Contains(s, "/") {
		var err error
		if p, err = netip.ParsePrefix(s); err != nil {
			return p, httpapi.InvalidInput(field, "enter a prefix such as 1.1.1.0/24 or an IP address")
		}
	} else {
		a, err := netip.ParseAddr(strings.Trim(s, "[]"))
		if err != nil || a.Zone() != "" {
			return p, httpapi.InvalidInput(field, "enter a prefix such as 1.1.1.0/24 or an IP address")
		}
		p = netip.PrefixFrom(a, a.BitLen())
	}
	p = p.Masked()
	a := p.Addr().Unmap()
	if !a.IsGlobalUnicast() || a.IsPrivate() || cgnat.Contains(a) || reserved4.Contains(a) {
		return p, httpapi.InvalidInput(field, "private and reserved address space is not routed on the Internet")
	}
	return p, nil
}
