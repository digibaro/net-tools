package email

import (
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// DKIMTool is EM-04.
type DKIMTool struct {
	deps tools.Deps
	res  resolver
}

func (*DKIMTool) ID() string                                { return "email.dkim" }
func (*DKIMTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *DKIMTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.res == nil {
		t.res = sysResolver{}
	}
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(15*time.Second)).Get("/email/dkim", t.handle)
}

// DKIMResult is the response payload.
type DKIMResult struct {
	Domain   string            `json:"domain"`
	Selector string            `json:"selector"`
	Name     string            `json:"name"`
	Record   string            `json:"record"`
	Tags     map[string]string `json:"tags"`
	KeyType  string            `json:"keyType"`
	KeyBits  int               `json:"keyBits"`
	Testing  bool              `json:"testing"`
	Revoked  bool              `json:"revoked"`
	Problems []string          `json:"problems"`
	Warnings []string          `json:"warnings"`
}

var selectorRe = regexp.MustCompile(`^[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*$`)

// CheckDKIM fetches and analyzes <selector>._domainkey.<domain>.
func CheckDKIM(ctx context.Context, res resolver, domain, selector string) DKIMResult {
	out := DKIMResult{Domain: domain, Selector: selector, Name: selector + "._domainkey." + domain, Tags: map[string]string{}, Problems: []string{}, Warnings: []string{}}
	txts, _, err := res.TXT(ctx, out.Name)
	if err != nil {
		out.Problems = append(out.Problems, "dns-error")
		return out
	}
	for _, t := range txts {
		if strings.Contains(t, "p=") {
			out.Record = t
			break
		}
	}
	if out.Record == "" {
		out.Problems = append(out.Problems, "no-record")
		return out
	}
	out.Tags = tagList(out.Record)
	if v, ok := out.Tags["v"]; ok && v != "DKIM1" {
		out.Problems = append(out.Problems, "bad-version")
	}
	out.KeyType = strings.ToLower(defaultTo(out.Tags["k"], "rsa"))
	out.Testing = strings.Contains(out.Tags["t"], "y")
	if out.Testing {
		out.Warnings = append(out.Warnings, "testing")
	}
	p := strings.Join(strings.Fields(out.Tags["p"]), "")
	if p == "" {
		out.Revoked = true
		out.Problems = append(out.Problems, "revoked")
		return out
	}
	der, err := base64.StdEncoding.DecodeString(p)
	if err != nil {
		out.Problems = append(out.Problems, "bad-key")
		return out
	}
	switch out.KeyType {
	case "ed25519":
		if len(der) == ed25519.PublicKeySize {
			out.KeyBits = 256
		} else {
			out.Problems = append(out.Problems, "bad-key")
		}
	default:
		pub, err := x509.ParsePKIXPublicKey(der)
		if err != nil {
			if k, err2 := x509.ParsePKCS1PublicKey(der); err2 == nil {
				pub = k
			} else {
				out.Problems = append(out.Problems, "bad-key")
				return out
			}
		}
		if k, ok := pub.(*rsa.PublicKey); ok {
			out.KeyBits = k.N.BitLen()
			switch {
			case out.KeyBits < 1024:
				out.Problems = append(out.Problems, "weak-key")
			case out.KeyBits < 2048:
				out.Warnings = append(out.Warnings, "short-key")
			}
		} else {
			out.Problems = append(out.Problems, "bad-key")
		}
	}
	return out
}

func (t *DKIMTool) handle(w http.ResponseWriter, r *http.Request) {
	domain, ok := domainParam(w, r, "domain")
	if !ok {
		return
	}
	q := params.New(r.URL.Query())
	selector := strings.ToLower(q.String("selector", "", true, 100))
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if !selectorRe.MatchString(selector) {
		httpapi.Error(w, r, httpapi.InvalidInput("selector", "a selector is a label such as google or s1"))
		return
	}
	httpapi.JSON(w, r, CheckDKIM(r.Context(), t.res, domain, selector), &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: domain}})
}
