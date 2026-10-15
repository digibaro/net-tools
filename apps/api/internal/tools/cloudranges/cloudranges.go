// Package cloudranges implements BGP-12: which cloud or CDN provider (and service/region)
// publishes a range containing an IP. Lists are fetched from the providers' official
// endpoints and kept parsed in memory for 24 h.
package cloudranges

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/cache"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/providers"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/reputation"
)

// TTL is how long a parsed list is kept.
const TTL = 24 * time.Hour

// Range is one published prefix.
type Range struct {
	Prefix   netip.Prefix
	Provider string
	Service  string
	Region   string
}

// Source is one provider list.
type Source struct {
	Name  string
	URL   string
	Parse func([]byte) ([]Range, error)
	// Discover finds the list URL in the page at URL (for lists published under changing names).
	Discover func(page []byte) (string, error)
	// MaxBytes raises the 5 MiB response cap for large lists.
	MaxBytes int64
}

// azureListRe matches the weekly Service Tags file linked from Microsoft's download page.
var azureListRe = regexp.MustCompile(`https://download\.microsoft\.com/download/[\w/.-]+/ServiceTags_Public_\d{8}\.json`)

// discoverAzure returns the current Service Tags URL; only download.microsoft.com links are used.
func discoverAzure(page []byte) (string, error) {
	u := azureListRe.Find(page)
	if u == nil {
		return "", errors.New("no Service Tags link on the download page")
	}
	return string(u), nil
}

func parseAzure(b []byte) ([]Range, error) {
	var d struct {
		Values []struct {
			Name       string `json:"name"`
			Properties struct {
				Region          string   `json:"region"`
				SystemService   string   `json:"systemService"`
				AddressPrefixes []string `json:"addressPrefixes"`
			} `json:"properties"`
		} `json:"values"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err //nolint:wrapcheck // classified by caller
	}
	if len(d.Values) == 0 {
		return nil, errors.New("no service tags")
	}
	type key struct {
		p               netip.Prefix
		service, region string
	}
	seen := map[key]bool{}
	var out []Range
	for _, v := range d.Values {
		service := v.Properties.SystemService
		if service == "" {
			service = strings.SplitN(v.Name, ".", 2)[0] // e.g. AzureCloud.westeurope
		}
		for _, s := range v.Properties.AddressPrefixes {
			pr, ok := p(s)
			if !ok {
				continue
			}
			k := key{pr, service, v.Properties.Region}
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, Range{pr, "Microsoft Azure", service, v.Properties.Region})
		}
	}
	return out, nil
}

func p(s string) (netip.Prefix, bool) {
	pr, err := netip.ParsePrefix(strings.TrimSpace(s))
	return pr.Masked(), err == nil
}

func parseAWS(b []byte) ([]Range, error) {
	var d struct {
		Prefixes []struct {
			P       string `json:"ip_prefix"`
			Region  string `json:"region"`
			Service string `json:"service"`
		} `json:"prefixes"`
		V6 []struct {
			P       string `json:"ipv6_prefix"`
			Region  string `json:"region"`
			Service string `json:"service"`
		} `json:"ipv6_prefixes"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err //nolint:wrapcheck // wrapped by caller
	}
	var out []Range
	for _, x := range d.Prefixes {
		if pr, ok := p(x.P); ok {
			out = append(out, Range{pr, "Amazon Web Services", x.Service, x.Region})
		}
	}
	for _, x := range d.V6 {
		if pr, ok := p(x.P); ok {
			out = append(out, Range{pr, "Amazon Web Services", x.Service, x.Region})
		}
	}
	return out, nil
}

func parseGCP(b []byte) ([]Range, error) {
	var d struct {
		Prefixes []struct {
			V4      string `json:"ipv4Prefix"`
			V6      string `json:"ipv6Prefix"`
			Service string `json:"service"`
			Scope   string `json:"scope"`
		} `json:"prefixes"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err //nolint:wrapcheck // wrapped by caller
	}
	var out []Range
	for _, x := range d.Prefixes {
		if pr, ok := p(x.V4 + x.V6); ok {
			out = append(out, Range{pr, "Google Cloud", x.Service, x.Scope})
		}
	}
	return out, nil
}

func cidrList(provider, service string, lists ...[]string) []Range {
	var out []Range
	for _, l := range lists {
		for _, s := range l {
			if pr, ok := p(s); ok {
				out = append(out, Range{pr, provider, service, ""})
			}
		}
	}
	return out
}

func parseCloudflare(b []byte) ([]Range, error) {
	var d struct {
		Result struct {
			V4 []string `json:"ipv4_cidrs"`
			V6 []string `json:"ipv6_cidrs"`
		} `json:"result"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err //nolint:wrapcheck // wrapped by caller
	}
	return cidrList("Cloudflare", "CDN / proxy", d.Result.V4, d.Result.V6), nil
}

func parseFastly(b []byte) ([]Range, error) {
	var d struct {
		V4 []string `json:"addresses"`
		V6 []string `json:"ipv6_addresses"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err //nolint:wrapcheck // wrapped by caller
	}
	return cidrList("Fastly", "CDN", d.V4, d.V6), nil
}

func parseGitHub(b []byte) ([]Range, error) {
	var d map[string]json.RawMessage
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err //nolint:wrapcheck // wrapped by caller
	}
	var out []Range
	for svc, raw := range d {
		var l []string
		if json.Unmarshal(raw, &l) == nil {
			out = append(out, cidrList("GitHub", svc, l)...)
		}
	}
	return out, nil
}

func parseOracle(b []byte) ([]Range, error) {
	type cidr struct {
		CIDR string   `json:"cidr"`
		Tags []string `json:"tags"`
	}
	var d struct {
		Regions []struct {
			Region string `json:"region"`
			CIDRs  []cidr `json:"cidrs"`
			V6     []cidr `json:"ipv6_cidrs"`
		} `json:"regions"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err //nolint:wrapcheck // wrapped by caller
	}
	var out []Range
	for _, r := range d.Regions {
		for _, c := range append(r.CIDRs, r.V6...) {
			if pr, ok := p(c.CIDR); ok {
				out = append(out, Range{pr, "Oracle Cloud", strings.Join(c.Tags, ", "), r.Region})
			}
		}
	}
	return out, nil
}

func parseDigitalOcean(b []byte) ([]Range, error) {
	var out []Range
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		f := strings.Split(sc.Text(), ",")
		if pr, ok := p(f[0]); ok {
			region := ""
			if len(f) >= 4 {
				region = strings.Trim(f[3]+", "+f[1], ", ")
			}
			out = append(out, Range{pr, "DigitalOcean", "Droplets", region})
		}
	}
	return out, nil
}

// DefaultSources are the official published lists.
var DefaultSources = []Source{
	{"Amazon Web Services", "https://ip-ranges.amazonaws.com/ip-ranges.json", parseAWS, nil, 0},
	{"Google Cloud", "https://www.gstatic.com/ipranges/cloud.json", parseGCP, nil, 0},
	{"Cloudflare", "https://api.cloudflare.com/client/v4/ips", parseCloudflare, nil, 0},
	{"Fastly", "https://api.fastly.com/public-ip-list", parseFastly, nil, 0},
	{"GitHub", "https://api.github.com/meta", parseGitHub, nil, 0},
	{"Oracle Cloud", "https://docs.oracle.com/en-us/iaas/tools/public_ip_ranges.json", parseOracle, nil, 0},
	{"DigitalOcean", "https://digitalocean.com/geo/google.csv", parseDigitalOcean, nil, 0},
	// The Service Tags file name changes weekly; the download page links the current one (~4.5 MB).
	{"Microsoft Azure", "https://www.microsoft.com/en-us/download/details.aspx?id=56519", parseAzure, discoverAzure, 16 << 20},
}

// Tool is bgp.cloudranges.
type Tool struct {
	deps    tools.Deps
	Sources []Source
	lists   *cache.TTL[[]Range]
}

func (*Tool) ID() string                                { return "bgp.cloudranges" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Sources == nil {
		t.Sources = DefaultSources
	}
	t.lists = cache.New[[]Range](len(t.Sources) + 1)
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(35*time.Second)).Get("/cloud-ranges", t.handle)
}

func (t *Tool) list(ctx context.Context, s Source) ([]Range, error) {
	return t.lists.Do(ctx, s.Name, TTL, func(ctx context.Context) ([]Range, error) {
		url := s.URL
		if s.Discover != nil {
			page, err := t.deps.Providers.Get(ctx, providers.Request{Provider: s.Name, URL: s.URL, TTL: -1, Timeout: 25 * time.Second, Header: http.Header{"Accept": {"text/html"}}})
			if err != nil {
				return nil, err //nolint:wrapcheck // client-safe
			}
			if url, err = s.Discover(page); err != nil {
				return nil, &providers.Error{Provider: s.Name, Message: s.Name + " changed its download page; the range list was not found", Err: err}
			}
		}
		b, err := t.deps.Providers.Get(ctx, providers.Request{Provider: s.Name, URL: url, TTL: -1, Timeout: 25 * time.Second, MaxBytes: s.MaxBytes, Header: http.Header{"Accept": {"*/*"}}})
		if err != nil {
			return nil, err //nolint:wrapcheck // client-safe
		}
		ranges, err := s.Parse(b)
		if err != nil {
			return nil, &providers.Error{Provider: s.Name, Message: s.Name + " published an unreadable range list", Err: err}
		}
		return ranges, nil
	})
}

// Match is a range containing the IP.
type Match struct {
	Provider string `json:"provider"`
	Prefix   string `json:"prefix"`
	Service  string `json:"service"`
	Region   string `json:"region"`
}

// SourceStatus reports per-list availability.
type SourceStatus struct {
	Name   string `json:"name"`
	Ranges int    `json:"ranges"`
	Error  string `json:"error"`
}

// Result is the BGP-12 payload.
type Result struct {
	IP       string         `json:"ip"`
	Provider string         `json:"provider"`
	Matches  []Match        `json:"matches"`
	Sources  []SourceStatus `json:"sources"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("ip", "", true, 64)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ip, err := reputation.PublicIP(raw)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out := Result{IP: ip.String(), Matches: []Match{}, Sources: make([]SourceStatus, len(t.Sources))}
	type hit struct {
		m    Match
		bits int
	}
	hits := make([][]hit, len(t.Sources))
	ctx := r.Context()
	var wg sync.WaitGroup
	for i, s := range t.Sources {
		wg.Go(func() {
			ranges, err := t.list(ctx, s)
			out.Sources[i] = SourceStatus{Name: s.Name, Ranges: len(ranges)}
			if err != nil {
				out.Sources[i].Error = httpapi.ToAPIError(err).Message
				return
			}
			for _, rg := range ranges {
				if rg.Prefix.Contains(ip) {
					hits[i] = append(hits[i], hit{Match{rg.Provider, rg.Prefix.String(), rg.Service, rg.Region}, rg.Prefix.Bits()})
				}
			}
		})
	}
	wg.Wait()
	var all []hit
	for _, h := range hits {
		all = append(all, h...)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].bits > all[j].bits }) // most specific first
	for _, h := range all {
		out.Matches = append(out.Matches, h.m)
	}
	if len(out.Matches) > 0 {
		out.Provider = out.Matches[0].Provider
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}
