package lan

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
)

// Ports used by the gateway protocols (RFC 6886 NAT-PMP, RFC 6887 PCP share 5351).
const natPMPPort = 5351

var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// NATPMPResult is what the gateway said over NAT-PMP / PCP.
type NATPMPResult struct {
	Supported  bool   `json:"supported"`
	ExternalIP string `json:"externalIp"`
	Epoch      uint32 `json:"epoch"`
	PCP        bool   `json:"pcp"`
	Error      string `json:"error"`
}

// UPnPResult is what the gateway said over UPnP IGD.
type UPnPResult struct {
	Supported    bool   `json:"supported"`
	ExternalIP   string `json:"externalIp"`
	FriendlyName string `json:"friendlyName"`
	Model        string `json:"model"`
	Service      string `json:"service"`
	Error        string `json:"error"`
}

// GatewayResult is the /lan/gateway response.
type GatewayResult struct {
	Interface  string       `json:"interface"`
	Gateway    string       `json:"gateway"`
	GatewayMAC string       `json:"gatewayMac"`
	Addresses  []string     `json:"addresses"`
	NATPMP     NATPMPResult `json:"natPmp"`
	UPnP       UPnPResult   `json:"upnp"`
	ExternalIP string       `json:"externalIp"`
	DoubleNAT  bool         `json:"doubleNat"`
}

// ParseNATPMP decodes a NAT-PMP external-address response (12 bytes).
func ParseNATPMP(b []byte) (netip.Addr, uint32, error) {
	if len(b) < 12 || b[0] != 0 || b[1] != 128 {
		if len(b) >= 2 && b[0] == 2 {
			return netip.Addr{}, 0, errPCPOnly
		}
		return netip.Addr{}, 0, errors.New("not a NAT-PMP response")
	}
	if code := binary.BigEndian.Uint16(b[2:4]); code != 0 {
		return netip.Addr{}, 0, fmt.Errorf("result code %d", code)
	}
	return netip.AddrFrom4([4]byte(b[8:12])), binary.BigEndian.Uint32(b[4:8]), nil
}

var errPCPOnly = errors.New("gateway speaks PCP only")

// pcpAnnounce builds a PCP ANNOUNCE request (creates no mapping).
func pcpAnnounce(client netip.Addr) []byte {
	b := make([]byte, 24)
	b[0], b[1] = 2, 0
	a := client.As16()
	copy(b[8:], a[:])
	return b
}

func udpExchange(ctx context.Context, gw netip.Addr, port int, req []byte) ([]byte, netip.Addr, error) {
	var d net.Dialer
	c, err := d.DialContext(ctx, "udp4", netip.AddrPortFrom(gw, uint16(port)).String()) //nolint:gosec // fixed port
	if err != nil {
		return nil, netip.Addr{}, fmt.Errorf("dial: %w", err)
	}
	defer c.Close()
	local := c.LocalAddr().(*net.UDPAddr).AddrPort().Addr().Unmap() //nolint:forcetypeassert // udp
	buf := make([]byte, 1100)
	for _, wait := range []time.Duration{250 * time.Millisecond, 500 * time.Millisecond, time.Second} {
		if _, err := c.Write(req); err != nil {
			return nil, local, fmt.Errorf("send: %w", err)
		}
		_ = c.SetReadDeadline(time.Now().Add(wait))
		n, err := c.Read(buf)
		if err == nil {
			return buf[:n], local, nil
		}
		if ctx.Err() != nil {
			break
		}
	}
	return nil, local, errors.New("no answer")
}

func natPMP(ctx context.Context, gw netip.Addr) NATPMPResult { return natPMPAt(ctx, gw, natPMPPort) }

func natPMPAt(ctx context.Context, gw netip.Addr, port int) NATPMPResult {
	var r NATPMPResult
	resp, local, err := udpExchange(ctx, gw, port, []byte{0, 0})
	if err != nil {
		r.Error = "no NAT-PMP/PCP answer from the gateway"
		return r
	}
	ip, epoch, err := ParseNATPMP(resp)
	switch {
	case err == nil:
		r.Supported, r.ExternalIP, r.Epoch = true, ip.String(), epoch
	case !errors.Is(err, errPCPOnly):
		r.Error = err.Error()
	}
	if pcp, _, err := udpExchange(ctx, gw, port, pcpAnnounce(local)); err == nil && len(pcp) >= 24 && pcp[0] == 2 && pcp[1] == 0x80 {
		r.PCP = true
	}
	return r
}

type igdDesc struct {
	URLBase string    `xml:"URLBase"`
	Device  igdDevice `xml:"device"`
}

type igdDevice struct {
	DeviceType   string `xml:"deviceType"`
	FriendlyName string `xml:"friendlyName"`
	ModelName    string `xml:"modelName"`
	Services     []struct {
		ServiceType string `xml:"serviceType"`
		ControlURL  string `xml:"controlURL"`
	} `xml:"serviceList>service"`
	Devices []igdDevice `xml:"deviceList>device"`
}

// FindWANService walks the IGD device tree for WANIPConnection / WANPPPConnection.
func FindWANService(d igdDevice) (serviceType, controlURL string) {
	for _, s := range d.Services {
		if strings.Contains(s.ServiceType, ":WANIPConnection:") || strings.Contains(s.ServiceType, ":WANPPPConnection:") {
			return s.ServiceType, s.ControlURL
		}
	}
	for _, c := range d.Devices {
		if st, cu := FindWANService(c); st != "" {
			return st, cu
		}
	}
	return "", ""
}

// ParseExternalIP reads NewExternalIPAddress from a SOAP response.
func ParseExternalIP(b []byte) (netip.Addr, error) {
	var env struct {
		IP string `xml:"Body>GetExternalIPAddressResponse>NewExternalIPAddress"`
	}
	if err := xml.Unmarshal(b, &env); err != nil { //nolint:gosec // capped response, no external entities
		return netip.Addr{}, fmt.Errorf("soap: %w", err)
	}
	a, err := netip.ParseAddr(strings.TrimSpace(env.IP))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("external IP: %w", err)
	}
	return a, nil
}

// onGateway resolves ref against base and requires the result to live on the gateway itself.
func onGateway(base, ref string, gw netip.Addr) (string, bool) {
	b, err := url.Parse(base)
	if err != nil {
		return "", false
	}
	u, err := b.Parse(ref)
	if err != nil || u.Scheme != "http" {
		return "", false
	}
	a, err := netip.ParseAddr(u.Hostname())
	return u.String(), err == nil && a.Unmap() == gw
}

// HTTPDoer performs gateway HTTP calls (injectable for tests).
type HTTPDoer func(ctx context.Context, method, u, soapAction string, body []byte) ([]byte, error)

func gatewayHTTP(ctx context.Context, method, u, soapAction string, body []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	if soapAction != "" {
		req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
		req.Header.Set("SOAPAction", soapAction)
	}
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(req) //nolint:gosec // URL host checked to be the default gateway
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return b, nil
}

const igdSearch = "M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 1\r\nST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\n\r\n"

func (t *GatewayTool) upnp(ctx context.Context, gw netip.Addr) UPnPResult {
	var r UPnPResult
	pc, err := t.Listen(ctx)
	if err != nil {
		r.Error = "cannot open a UDP socket"
		return r
	}
	defer pc.Close()
	_, _ = pc.WriteTo([]byte(igdSearch), net.UDPAddrFromAddrPort(ssdpGroup))
	loc := ""
	buf := make([]byte, 4096)
	deadline := time.Now().Add(2 * time.Second)
	for loc == "" && time.Now().Before(deadline) && ctx.Err() == nil {
		_ = pc.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		n, from, err := pc.ReadFrom(buf)
		if err != nil {
			continue
		}
		ua, ok := from.(*net.UDPAddr)
		if !ok || ua.AddrPort().Addr().Unmap() != gw {
			continue // only the default gateway's answer counts
		}
		if h, err := ParseSSDPResponse(buf[:n]); err == nil {
			if u, ok := onGateway(h["LOCATION"], "", gw); ok {
				loc = u
			}
		}
	}
	if loc == "" {
		r.Error = "the gateway did not answer the UPnP IGD search"
		return r
	}
	descXML, err := t.HTTP(ctx, http.MethodGet, loc, "", nil)
	if err != nil {
		r.Error = "IGD description unavailable"
		return r
	}
	var desc igdDesc
	if xml.Unmarshal(descXML, &desc) != nil { //nolint:gosec // capped, no external entities
		r.Error = "unreadable IGD description"
		return r
	}
	r.Supported, r.FriendlyName, r.Model = true, desc.Device.FriendlyName, desc.Device.ModelName
	st, ctl := FindWANService(desc.Device)
	if st == "" {
		r.Error = "no WANIPConnection service"
		return r
	}
	r.Service = st
	ctlURL, ok := onGateway(baseURL(desc.URLBase, loc), ctl, gw)
	if !ok {
		r.Error = "control URL is not on the gateway; not called"
		return r
	}
	soap := []byte(`<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:GetExternalIPAddress xmlns:u="` + st + `"/></s:Body></s:Envelope>`)
	resp, err := t.HTTP(ctx, http.MethodPost, ctlURL, `"`+st+`#GetExternalIPAddress"`, soap)
	if err != nil {
		r.Error = "GetExternalIPAddress failed"
		return r
	}
	ip, err := ParseExternalIP(resp)
	if err != nil {
		r.Error = "GetExternalIPAddress returned no address"
		return r
	}
	r.ExternalIP = ip.String()
	return r
}

func baseURL(urlBase, loc string) string {
	if strings.TrimSpace(urlBase) != "" {
		return strings.TrimSpace(urlBase)
	}
	return loc
}

// GatewayTool is LAN-06.
type GatewayTool struct {
	deps     tools.Deps
	ReadFile func(string) ([]byte, error)
	Addrs    func(iface string) []string
	NATPMP   func(ctx context.Context, gw netip.Addr) NATPMPResult
	Listen   func(ctx context.Context) (PacketConn, error)
	HTTP     HTTPDoer
}

func (*GatewayTool) ID() string                                { return "lan.gateway" }
func (*GatewayTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.RequireLAN(d) }

func ifaceAddrs(name string) []string {
	out := []string{}
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return out
	}
	addrs, _ := ifi.Addrs()
	for _, a := range addrs {
		out = append(out, a.String())
	}
	return out
}

func (t *GatewayTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.ReadFile == nil {
		t.ReadFile = os.ReadFile
	}
	if t.Addrs == nil {
		t.Addrs = ifaceAddrs
	}
	if t.NATPMP == nil {
		t.NATPMP = natPMP
	}
	if t.Listen == nil {
		t.Listen = listenUDP4
	}
	if t.HTTP == nil {
		t.HTTP = gatewayHTTP
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(15*time.Second)).Get("/lan/gateway", t.handle)
}

func (t *GatewayTool) handle(w http.ResponseWriter, r *http.Request) {
	b, _ := t.ReadFile("/proc/net/route")
	routes := ParseDefaultRoutes(b)
	if len(routes) == 0 || !routes[0].Gateway.IsValid() || routes[0].Gateway.IsUnspecified() {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeInternal, Message: "no IPv4 default gateway found on the NetTools server"})
		return
	}
	rt := routes[0]
	res := GatewayResult{Interface: rt.Interface, Gateway: rt.Gateway.String(), Addresses: t.Addrs(rt.Interface)}
	if arpTab, err := t.ReadFile("/proc/net/arp"); err == nil {
		res.GatewayMAC = ParseARPTable(arpTab)[rt.Gateway]
	}
	ctx := r.Context()
	pmpDone := make(chan NATPMPResult, 1)
	go func() { pmpDone <- t.NATPMP(ctx, rt.Gateway) }()
	res.UPnP = t.upnp(ctx, rt.Gateway)
	res.NATPMP = <-pmpDone
	res.ExternalIP = res.NATPMP.ExternalIP
	if res.ExternalIP == "" {
		res.ExternalIP = res.UPnP.ExternalIP
	}
	if a, err := netip.ParseAddr(res.ExternalIP); err == nil {
		res.DoubleNAT = a.IsPrivate() || cgnat.Contains(a)
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}
