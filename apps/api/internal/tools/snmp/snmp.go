// Package snmp implements CON-12: bounded SNMP v2c/v3 get and walk (gosnmp).
// Off unless SNMP_ENABLED. Credentials travel in the POST body and are never logged.
package snmp

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/gosnmp/gosnmp"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
)

// Limits.
const (
	MaxRows    = 1000
	MaxGetOIDs = 20
	reqTimeout = 2 * time.Second
)

// Tool is net.snmp.
type Tool struct {
	deps tools.Deps
	// Redirect rewrites the agent address (tests only; nil in production).
	Redirect func(netip.AddrPort) netip.AddrPort
}

func (*Tool) ID() string { return "net.snmp" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) {
	return tools.RequireFlag(d.Config.Features.SNMP)
}

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(25*time.Second)).Post("/snmp", t.handle)
}

// V3 holds USM credentials.
type V3 struct {
	User         string `json:"user"`
	Level        string `json:"level"` // noAuthNoPriv | authNoPriv | authPriv
	AuthProtocol string `json:"authProtocol"`
	AuthPassword string `json:"authPassword"`
	PrivProtocol string `json:"privProtocol"`
	PrivPassword string `json:"privPassword"`
	Context      string `json:"context"`
}

// Request is the POST body.
type Request struct {
	Target    string   `json:"target"`
	Port      int      `json:"port"`
	Version   string   `json:"version"` // 2c | 3
	Community string   `json:"community"`
	V3        *V3      `json:"v3"`
	Operation string   `json:"operation"` // get | walk
	OIDs      []string `json:"oids"`
	MaxRows   int      `json:"maxRows"`
}

// Var is one varbind.
type Var struct {
	OID   string `json:"oid"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Result is the response payload.
type Result struct {
	Address   string `json:"address"`
	Version   string `json:"version"`
	Operation string `json:"operation"`
	Vars      []Var  `json:"vars"`
	Truncated bool   `json:"truncated"`
}

var oidRe = regexp.MustCompile(`^\.?[0-2](\.\d{1,10})+$`)

// NormalizeOID accepts dotted numeric OIDs with or without the leading dot.
func NormalizeOID(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) > 400 || !oidRe.MatchString(s) {
		return "", false
	}
	return "." + strings.TrimPrefix(s, "."), true
}

var authProtos = map[string]gosnmp.SnmpV3AuthProtocol{"MD5": gosnmp.MD5, "SHA": gosnmp.SHA, "SHA224": gosnmp.SHA224, "SHA256": gosnmp.SHA256, "SHA384": gosnmp.SHA384, "SHA512": gosnmp.SHA512}

var privProtos = map[string]gosnmp.SnmpV3PrivProtocol{"DES": gosnmp.DES, "AES": gosnmp.AES, "AES192": gosnmp.AES192, "AES256": gosnmp.AES256, "AES192C": gosnmp.AES192C, "AES256C": gosnmp.AES256C}

// Validate checks the request and returns the normalized OIDs.
func (req *Request) Validate() ([]string, error) {
	if req.Port == 0 {
		req.Port = 161
	}
	if req.Port < 1 || req.Port > 65535 {
		return nil, httpapi.InvalidInput("port", "port must be 1-65535")
	}
	if req.MaxRows == 0 {
		req.MaxRows = 200
	}
	if req.MaxRows < 1 || req.MaxRows > MaxRows {
		return nil, httpapi.InvalidInput("maxRows", "maxRows must be 1-"+strconv.Itoa(MaxRows))
	}
	switch req.Version {
	case "", "2c":
		req.Version = "2c"
		if req.Community == "" || len(req.Community) > 64 {
			return nil, httpapi.InvalidInput("community", "enter the community string (at most 64 characters)")
		}
	case "3":
		if err := req.V3.validate(); err != nil {
			return nil, err
		}
	default:
		return nil, httpapi.InvalidInput("version", "version must be 2c or 3")
	}
	if req.Operation != "get" && req.Operation != "walk" {
		return nil, httpapi.InvalidInput("operation", "operation must be get or walk")
	}
	if len(req.OIDs) == 0 || req.Operation == "walk" && len(req.OIDs) != 1 || len(req.OIDs) > MaxGetOIDs {
		return nil, httpapi.InvalidInput("oids", "give one OID to walk, or up to "+strconv.Itoa(MaxGetOIDs)+" to get")
	}
	out := make([]string, 0, len(req.OIDs))
	for _, o := range req.OIDs {
		n, ok := NormalizeOID(o)
		if !ok {
			return nil, httpapi.InvalidInput("oids", "OIDs must be numeric, such as 1.3.6.1.2.1.1")
		}
		out = append(out, n)
	}
	return out, nil
}

func (v *V3) validate() error {
	if v == nil || v.User == "" || len(v.User) > 64 {
		return httpapi.InvalidInput("v3.user", "enter the SNMPv3 user name")
	}
	switch v.Level {
	case "noAuthNoPriv":
	case "authNoPriv", "authPriv":
		if _, ok := authProtos[v.AuthProtocol]; !ok {
			return httpapi.InvalidInput("v3.authProtocol", "unknown authentication protocol")
		}
		if len(v.AuthPassword) < 8 || len(v.AuthPassword) > 128 {
			return httpapi.InvalidInput("v3.authPassword", "SNMPv3 passwords are 8-128 characters")
		}
		if v.Level == "authPriv" {
			if _, ok := privProtos[v.PrivProtocol]; !ok {
				return httpapi.InvalidInput("v3.privProtocol", "unknown privacy protocol")
			}
			if len(v.PrivPassword) < 8 || len(v.PrivPassword) > 128 {
				return httpapi.InvalidInput("v3.privPassword", "SNMPv3 passwords are 8-128 characters")
			}
		}
	default:
		return httpapi.InvalidInput("v3.level", "level must be noAuthNoPriv, authNoPriv or authPriv")
	}
	return nil
}

// Client builds the gosnmp client for addr.
func (req *Request) Client(addr netip.AddrPort) *gosnmp.GoSNMP {
	g := &gosnmp.GoSNMP{
		Target: addr.Addr().String(), Port: addr.Port(), Transport: "udp", Timeout: reqTimeout, Retries: 1,
		MaxOids: MaxGetOIDs, MaxRepetitions: 25, Version: gosnmp.Version2c, Community: req.Community,
	}
	if req.Version == "3" {
		v := req.V3
		sp := &gosnmp.UsmSecurityParameters{UserName: v.User, AuthenticationProtocol: gosnmp.NoAuth, PrivacyProtocol: gosnmp.NoPriv}
		g.MsgFlags = gosnmp.NoAuthNoPriv
		if v.Level != "noAuthNoPriv" {
			sp.AuthenticationProtocol, sp.AuthenticationPassphrase = authProtos[v.AuthProtocol], v.AuthPassword
			g.MsgFlags = gosnmp.AuthNoPriv
		}
		if v.Level == "authPriv" {
			sp.PrivacyProtocol, sp.PrivacyPassphrase = privProtos[v.PrivProtocol], v.PrivPassword
			g.MsgFlags = gosnmp.AuthPriv
		}
		g.Version, g.Community, g.SecurityModel, g.SecurityParameters, g.ContextName = gosnmp.Version3, "", gosnmp.UserSecurityModel, sp, v.Context
	}
	return g
}

// Format renders one PDU.
func Format(p gosnmp.SnmpPDU) Var {
	v := Var{OID: p.Name, Type: p.Type.String()}
	switch p.Type {
	case gosnmp.OctetString:
		b, _ := p.Value.([]byte)
		if utf8.Valid(b) && printable(b) {
			v.Value = string(b)
		} else {
			v.Type, v.Value = "Hex-STRING", strings.ToUpper(hex.EncodeToString(b))
		}
	case gosnmp.ObjectIdentifier, gosnmp.IPAddress:
		v.Value = fmt.Sprint(p.Value)
	case gosnmp.Null, gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.EndOfMibView:
		v.Value = ""
	default:
		v.Value = gosnmp.ToBigInt(p.Value).String()
	}
	return v
}

func printable(b []byte) bool {
	for _, r := range string(b) {
		if r < 0x20 && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

var errEnough = errors.New("row limit reached")

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("body", "invalid JSON"))
		return
	}
	oids, err := req.Validate()
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	tgt, err := t.deps.Target.Resolve(r.Context(), req.Target, target.Opts{Field: "target"})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	addr := netip.AddrPortFrom(tgt.IP, uint16(req.Port)) //nolint:gosec // 1..65535 validated
	if t.Redirect != nil {
		addr = t.Redirect(addr)
	}
	g := req.Client(addr)
	g.Context = r.Context()
	if err := g.Connect(); err != nil {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeInternal, Message: "cannot open a UDP socket", Err: err})
		return
	}
	defer g.Close()
	res := Result{Address: tgt.IP.String(), Version: req.Version, Operation: req.Operation, Vars: []Var{}}
	if req.Operation == "get" {
		var pkt *gosnmp.SnmpPacket
		pkt, err = g.Get(oids)
		if err == nil {
			for _, p := range pkt.Variables {
				res.Vars = append(res.Vars, Format(p))
			}
		}
	} else {
		err = g.BulkWalk(oids[0], func(p gosnmp.SnmpPDU) error {
			if len(res.Vars) >= req.MaxRows {
				res.Truncated = true
				return errEnough
			}
			res.Vars = append(res.Vars, Format(p))
			return nil
		})
		if errors.Is(err, errEnough) {
			err = nil
		}
	}
	if err != nil {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "the agent did not answer (" + agentError(err) + ")", Err: err})
		return
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}

// agentError turns gosnmp errors into a short reason without echoing credentials.
func agentError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "Request timeout"):
		return "timeout: wrong address, port, community or credentials, or SNMP blocked"
	case strings.Contains(msg, "unknown user"), strings.Contains(msg, "UnknownUserName"):
		return "unknown SNMPv3 user"
	case strings.Contains(msg, "wrong digest"), strings.Contains(msg, "WrongDigest"), strings.Contains(msg, "authentication"):
		return "SNMPv3 authentication failed"
	case strings.Contains(msg, "decrypt"):
		return "SNMPv3 decryption failed (privacy settings)"
	}
	return "SNMP error"
}
