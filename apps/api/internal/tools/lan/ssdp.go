package lan

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/stream"
)

var ssdpGroup = netip.MustParseAddrPort("239.255.255.250:1900")

// MaxDevices bounds the SSDP result.
const MaxDevices = 200

// DescFetcher fetches a device description (injectable for tests).
type DescFetcher func(ctx context.Context, u string) ([]byte, error)

// fetchDescription GETs the description without following redirects, capped at 64 KiB.
func fetchDescription(ctx context.Context, u string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(req) //nolint:gosec // URL host checked against the responder's own LAN address
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return b, nil
}

// SSDPDevice is the "device" event.
type SSDPDevice struct {
	Address      string   `json:"address"`
	Location     string   `json:"location"`
	Server       string   `json:"server"`
	USN          string   `json:"usn"`
	Types        []string `json:"types"`
	FriendlyName string   `json:"friendlyName"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
	DeviceType   string   `json:"deviceType"`
	Presentation string   `json:"presentationUrl"`
	Error        string   `json:"error"`
}

// SSDPTool is LAN-04.
type SSDPTool struct {
	deps   tools.Deps
	Listen func(ctx context.Context) (PacketConn, error)
	Fetch  DescFetcher
}

func (*SSDPTool) ID() string                                { return "lan.ssdp" }
func (*SSDPTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.RequireLAN(d) }

func (t *SSDPTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Listen == nil {
		t.Listen = listenUDP4
	}
	if t.Fetch == nil {
		t.Fetch = fetchDescription
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(25*time.Second)).Get("/lan/ssdp/stream", t.handle)
}

const mSearch = "M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 2\r\nST: ssdp:all\r\nUSER-AGENT: NetTools/1 UPnP/1.1\r\n\r\n"

// ParseSSDPResponse reads the HTTP-like reply headers.
func ParseSSDPResponse(b []byte) (map[string]string, error) {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(b)), nil)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	_ = resp.Body.Close()
	out := map[string]string{}
	for k := range resp.Header {
		out[strings.ToUpper(k)] = resp.Header.Get(k)
	}
	return out, nil
}

type deviceDesc struct {
	Device struct {
		DeviceType   string `xml:"deviceType"`
		FriendlyName string `xml:"friendlyName"`
		Manufacturer string `xml:"manufacturer"`
		ModelName    string `xml:"modelName"`
		ModelNumber  string `xml:"modelNumber"`
		Presentation string `xml:"presentationURL"`
	} `xml:"device"`
}

// locationAllowed: the description must live on the responder itself (no third-party fetches).
func locationAllowed(loc string, from netip.Addr) bool {
	u, err := url.Parse(loc)
	if err != nil || u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	a, err := netip.ParseAddr(u.Hostname())
	return err == nil && a.Unmap() == from.Unmap()
}

func (t *SSDPTool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		seconds := q.Int("seconds", 4, 1, 10)
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		pc, err := t.Listen(ctx)
		if err != nil {
			return nil, fmt.Errorf("udp socket: %w", err)
		}
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "seconds": seconds}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			defer pc.Close()
			return t.discover(ctx, pc, time.Duration(seconds)*time.Second, emit), nil
		}}, nil
	})
}

// SSDPSummary is the done payload.
type SSDPSummary struct {
	Devices []SSDPDevice `json:"devices"`
}

func (t *SSDPTool) discover(ctx context.Context, pc PacketConn, dur time.Duration, emit stream.Emit) SSDPSummary {
	dst := net.UDPAddrFromAddrPort(ssdpGroup)
	_, _ = pc.WriteTo([]byte(mSearch), dst)
	devices := map[string]*SSDPDevice{} // by location (one description per device)
	var order []string
	deadline := time.Now().Add(dur)
	resent := false
	buf := make([]byte, 4096)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		if !resent && time.Until(deadline) < dur/2 {
			_, _ = pc.WriteTo([]byte(mSearch), dst) // UDP is lossy: ask twice
			resent = true
		}
		_ = pc.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		n, from, err := pc.ReadFrom(buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			break
		}
		h, err := ParseSSDPResponse(buf[:n])
		if err != nil || h["LOCATION"] == "" {
			continue
		}
		ua, _ := from.(*net.UDPAddr)
		src, _ := netip.AddrFromSlice(ua.IP)
		key := h["LOCATION"]
		d, ok := devices[key]
		if !ok {
			if len(devices) >= MaxDevices {
				continue
			}
			d = &SSDPDevice{Address: src.Unmap().String(), Location: key, Server: h["SERVER"], USN: h["USN"], Types: []string{}}
			devices[key] = d
			order = append(order, key)
		}
		if st := h["ST"]; st != "" && !slices.Contains(d.Types, st) && len(d.Types) < 30 {
			d.Types = append(d.Types, st)
		}
	}
	// Fetch descriptions in parallel, then emit each device once.
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, key := range order {
		d := devices[key]
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			src, _ := netip.ParseAddr(d.Address)
			if !locationAllowed(d.Location, src) {
				d.Error = "description is hosted elsewhere; not fetched"
				return
			}
			b, err := t.Fetch(ctx, d.Location)
			if err != nil {
				d.Error = "description unavailable"
				return
			}
			var desc deviceDesc
			if xml.Unmarshal(b, &desc) != nil { //nolint:gosec // 64 KiB cap; encoding/xml expands no external entities
				d.Error = "unreadable description"
				return
			}
			dv := desc.Device
			d.FriendlyName, d.Manufacturer, d.DeviceType, d.Presentation = dv.FriendlyName, dv.Manufacturer, dv.DeviceType, dv.Presentation
			d.Model = strings.TrimSpace(dv.ModelName + " " + dv.ModelNumber)
		})
	}
	wg.Wait()
	sum := SSDPSummary{Devices: []SSDPDevice{}}
	for _, key := range order {
		emit("device", *devices[key])
		sum.Devices = append(sum.Devices, *devices[key])
	}
	return sum
}
