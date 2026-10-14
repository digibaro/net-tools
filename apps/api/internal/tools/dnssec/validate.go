// Package dnssec implements DNS-05: validate the DNSSEC chain of trust for a name, from the
// root trust anchors down to the answer, and show where it breaks.
package dnssec

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/dnsx"
)

// RootAnchors are the IANA root zone KSK trust anchors (KSK-2017 and KSK-2024), checked against
// the live root DNSKEY set in October 2026.
var RootAnchors = []*dns.DS{
	{Hdr: dns.RR_Header{Name: ".", Rrtype: dns.TypeDS, Class: dns.ClassINET}, KeyTag: 20326, Algorithm: dns.RSASHA256, DigestType: dns.SHA256, Digest: "E06D44B80B8F1D39A95C0B0D7C65D08458E880409BBC683457104237C7F8EC8D"},
	{Hdr: dns.RR_Header{Name: ".", Rrtype: dns.TypeDS, Class: dns.ClassINET}, KeyTag: 38696, Algorithm: dns.RSASHA256, DigestType: dns.SHA256, Digest: "683D2D0ACB8C9B712A1948B27F741219298D0A450D612C483AF444A4C0FB2B16"},
}

// Status values.
const (
	Secure        = "secure"
	Insecure      = "insecure"
	Bogus         = "bogus"
	Indeterminate = "indeterminate"
)

// Key describes a DNSKEY.
type Key struct {
	Tag       uint16 `json:"tag"`
	Flags     uint16 `json:"flags"`
	Role      string `json:"role"`
	Algorithm string `json:"algorithm"`
	Trusted   bool   `json:"trusted"`
}

// DS describes a delegation signer record and whether a DNSKEY matches it.
type DS struct {
	Tag        uint16 `json:"tag"`
	Algorithm  string `json:"algorithm"`
	DigestType string `json:"digestType"`
	Matched    bool   `json:"matched"`
}

// Signature is one RRSIG check.
type Signature struct {
	Covers     string `json:"covers"`
	KeyTag     uint16 `json:"keyTag"`
	Algorithm  string `json:"algorithm"`
	Inception  string `json:"inception"`
	Expiration string `json:"expiration"`
	Valid      bool   `json:"valid"`
	Error      string `json:"error"`
}

// Step is one zone in the chain.
type Step struct {
	Zone       string      `json:"zone"`
	Status     string      `json:"status"`
	Message    string      `json:"message"`
	DS         []DS        `json:"ds"`
	Keys       []Key       `json:"keys"`
	Signatures []Signature `json:"signatures"`
}

// Answer is the final RRset (or denial of existence) and its signatures.
type Answer struct {
	Rcode      string        `json:"rcode"`
	Records    []dnsx.Record `json:"records"`
	Signatures []Signature   `json:"signatures"`
	Denial     string        `json:"denial"`
	Status     string        `json:"status"`
	Message    string        `json:"message"`
}

// Result is the DNS-05 payload.
type Result struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	ResolverAD bool   `json:"resolverAd"`
	Steps      []Step `json:"steps"`
	Answer     Answer `json:"answer"`
}

// Validator fetches records through one recursive resolver with DO and CD set (so bogus data is
// returned for inspection) and validates them itself.
type Validator struct {
	Server  dnsx.Server
	Anchors []*dns.DS
	Now     func() time.Time
}

func (v *Validator) query(ctx context.Context, name string, qt uint16, cd bool) (*dns.Msg, error) {
	m := dnsx.NewQuery(name, qt, true)
	m.CheckingDisabled = cd
	r, _, err := dnsx.Exchange(ctx, v.Server, m)
	if err != nil {
		return nil, fmt.Errorf("query %s %s: %w", name, dns.TypeToString[qt], err)
	}
	return r, nil
}

func rrsetOf(rrs []dns.RR, name string, qt uint16) ([]dns.RR, []*dns.RRSIG) {
	var set []dns.RR
	var sigs []*dns.RRSIG
	for _, rr := range rrs {
		h := rr.Header()
		if !strings.EqualFold(h.Name, name) {
			continue
		}
		if s, ok := rr.(*dns.RRSIG); ok {
			if s.TypeCovered == qt {
				sigs = append(sigs, s)
			}
			continue
		}
		if h.Rrtype == qt {
			set = append(set, rr)
		}
	}
	return set, sigs
}

func algName(a uint8) string {
	if s, ok := dns.AlgorithmToString[a]; ok {
		return s
	}
	return fmt.Sprint(a)
}

func digestName(d uint8) string {
	if s, ok := dns.HashToString[d]; ok {
		return s
	}
	return fmt.Sprint(d)
}

// verify checks sigs over set with keys (signer must be zone); at least one must be valid.
func (v *Validator) verify(set []dns.RR, sigs []*dns.RRSIG, keys []*dns.DNSKEY, zone string) ([]Signature, bool) {
	out := []Signature{}
	ok := false
	now := v.Now()
	for _, s := range sigs {
		sig := Signature{Covers: dns.TypeToString[s.TypeCovered], KeyTag: s.KeyTag, Algorithm: algName(s.Algorithm), Inception: dns.TimeToString(s.Inception), Expiration: dns.TimeToString(s.Expiration)}
		switch {
		case !strings.EqualFold(s.SignerName, zone):
			sig.Error = "signed by " + s.SignerName + ", not " + zone
		case !s.ValidityPeriod(now):
			sig.Error = "outside its validity period"
		default:
			sig.Error = "no matching key (tag " + fmt.Sprint(s.KeyTag) + ")"
			for _, k := range keys {
				if k.KeyTag() != s.KeyTag || k.Algorithm != s.Algorithm {
					continue
				}
				if err := s.Verify(k, set); err != nil {
					sig.Error = "signature does not verify: " + err.Error()
					continue
				}
				sig.Valid, sig.Error = true, ""
				break
			}
		}
		ok = ok || sig.Valid
		out = append(out, sig)
	}
	return out, ok
}

// zones returns the zone apexes from the root down to the zone holding name.
func (v *Validator) zones(ctx context.Context, name string) ([]string, error) {
	labels := dns.SplitDomainName(name)
	out := []string{"."}
	for i := len(labels) - 1; i >= 0; i-- {
		suffix := dns.Fqdn(strings.Join(labels[i:], "."))
		r, err := v.query(ctx, suffix, dns.TypeSOA, true)
		if err != nil {
			return nil, err
		}
		if soa, _ := rrsetOf(r.Answer, suffix, dns.TypeSOA); len(soa) > 0 {
			out = append(out, suffix)
		}
		if r.Rcode == dns.RcodeNameError {
			break
		}
	}
	return out, nil
}

// Validate walks the chain for name/qtype.
func (v *Validator) Validate(ctx context.Context, name string, qt uint16) (Result, error) {
	name = dns.Fqdn(strings.ToLower(name))
	res := Result{Name: name, Type: dns.TypeToString[qt], Steps: []Step{}}
	if r, err := v.query(ctx, name, qt, false); err == nil {
		res.ResolverAD = r.AuthenticatedData
	}
	zones, err := v.zones(ctx, name)
	if err != nil {
		return res, err
	}
	chain := Secure // state of the chain above the current zone
	var parentKeys []*dns.DNSKEY
	var zoneKeys []*dns.DNSKEY
	for i, zone := range zones {
		step := Step{Zone: zone, DS: []DS{}, Keys: []Key{}, Signatures: []Signature{}}
		if chain != Secure {
			step.Status, step.Message = chain, "not checked: the chain above is "+chain
			res.Steps = append(res.Steps, step)
			continue
		}
		// 1. The DS set: trust anchors for the root, else from the parent zone (signed there).
		var dsSet []*dns.DS
		if i == 0 {
			dsSet = v.Anchors
		} else {
			r, err := v.query(ctx, zone, dns.TypeDS, true)
			if err != nil {
				return res, err
			}
			set, sigs := rrsetOf(r.Answer, zone, dns.TypeDS)
			if len(set) == 0 {
				// No DS: an unsigned delegation (insecure) if the parent proves it with NSEC/NSEC3.
				step.Status, step.Message = Insecure, "no DS record at the parent: this zone is not signed (or its keys were never published to the parent)"
				if !hasDenial(r) {
					step.Message += "; the parent did not prove the absence of DS"
				}
				chain = Insecure
				res.Steps = append(res.Steps, step)
				continue
			}
			sigRes, ok := v.verify(set, sigs, parentKeys, zones[i-1])
			step.Signatures = append(step.Signatures, sigRes...)
			if !ok {
				step.Status, step.Message, chain = Bogus, "the DS records are not validly signed by the parent zone", Bogus
				res.Steps = append(res.Steps, step)
				continue
			}
			for _, rr := range set {
				dsSet = append(dsSet, rr.(*dns.DS)) //nolint:forcetypeassert // filtered by type
			}
		}
		// 2. The zone's DNSKEY set, signed by a key that matches a DS.
		r, err := v.query(ctx, zone, dns.TypeDNSKEY, true)
		if err != nil {
			return res, err
		}
		set, sigs := rrsetOf(r.Answer, zone, dns.TypeDNSKEY)
		var keys []*dns.DNSKEY
		for _, rr := range set {
			keys = append(keys, rr.(*dns.DNSKEY)) //nolint:forcetypeassert // filtered by type
		}
		var trusted []*dns.DNSKEY
		for _, ds := range dsSet {
			d := DS{Tag: ds.KeyTag, Algorithm: algName(ds.Algorithm), DigestType: digestName(ds.DigestType)}
			for _, k := range keys {
				if k.KeyTag() == ds.KeyTag && k.Algorithm == ds.Algorithm && strings.EqualFold(k.ToDS(ds.DigestType).Digest, ds.Digest) {
					d.Matched = true
					trusted = append(trusted, k)
				}
			}
			step.DS = append(step.DS, d)
		}
		for _, k := range keys {
			role := "ZSK"
			if k.Flags&dns.SEP != 0 {
				role = "KSK"
			}
			kd := Key{Tag: k.KeyTag(), Flags: k.Flags, Role: role, Algorithm: algName(k.Algorithm)}
			for _, tk := range trusted {
				kd.Trusted = kd.Trusted || tk == k
			}
			step.Keys = append(step.Keys, kd)
		}
		switch {
		case len(keys) == 0:
			step.Status, step.Message, chain = Bogus, "the parent has a DS record but the zone publishes no DNSKEY", Bogus
		case len(trusted) == 0:
			step.Status, step.Message, chain = Bogus, "no DNSKEY matches the DS records (a key rollover may have gone wrong)", Bogus
		default:
			sigRes, ok := v.verify(set, sigs, trusted, zone)
			step.Signatures = append(step.Signatures, sigRes...)
			if ok {
				step.Status, step.Message = Secure, "the DNSKEY set is signed by a key that matches the DS"
				parentKeys, zoneKeys = keys, keys
			} else {
				step.Status, step.Message, chain = Bogus, "the DNSKEY set is not validly signed by a key that matches the DS", Bogus
			}
		}
		res.Steps = append(res.Steps, step)
	}

	// 3. The answer itself.
	r, err := v.query(ctx, name, qt, true)
	if err != nil {
		return res, err
	}
	ans := Answer{Rcode: dnsx.RcodeName(r.Rcode), Records: []dnsx.Record{}, Signatures: []Signature{}}
	zone := zones[len(zones)-1]
	set, sigs := rrsetOf(r.Answer, name, qt)
	if len(set) == 0 {
		// CNAME at the name answers any type.
		if cn, cs := rrsetOf(r.Answer, name, dns.TypeCNAME); len(cn) > 0 {
			set, sigs = cn, cs
		}
	}
	ans.Records = dnsx.Records(set)
	switch {
	case chain != Secure:
		ans.Status, ans.Message = chain, "not validated: the chain is "+chain
	case len(set) > 0:
		sigRes, ok := v.verify(set, sigs, zoneKeys, zone)
		ans.Signatures = sigRes
		if ok {
			ans.Status, ans.Message = Secure, "the answer is signed by the zone"
		} else {
			ans.Status, ans.Message = Bogus, "the answer is not validly signed by the zone"
		}
	default:
		ans.Denial = denialKind(r)
		var all []Signature
		ok := ans.Denial != ""
		for _, t := range []uint16{dns.TypeNSEC, dns.TypeNSEC3} {
			byOwner := map[string][]dns.RR{}
			for _, rr := range r.Ns {
				if rr.Header().Rrtype == t {
					byOwner[rr.Header().Name] = append(byOwner[rr.Header().Name], rr)
				}
			}
			for owner, nset := range byOwner {
				_, nsigs := rrsetOf(r.Ns, owner, t)
				s, valid := v.verify(nset, nsigs, zoneKeys, zone)
				all = append(all, s...)
				ok = ok && valid
			}
		}
		ans.Signatures = all
		if ok {
			ans.Status, ans.Message = Secure, "the name or type does not exist, and the zone proves it ("+ans.Denial+")"
		} else {
			ans.Status, ans.Message = Bogus, "no validly signed proof of non-existence"
		}
	}
	res.Answer = ans
	res.Status = ans.Status
	switch res.Status {
	case Insecure:
		res.Message = "the name is in an unsigned zone"
	case Bogus:
		res.Message = "validation fails: validating resolvers will answer SERVFAIL"
	default:
		res.Message = "the answer validates from the root trust anchor"
	}
	for _, s := range res.Steps {
		if s.Status == Bogus || s.Status == Insecure {
			res.Message += " (at " + s.Zone + ")"
			break
		}
	}
	return res, nil
}

func hasDenial(r *dns.Msg) bool { return denialKind(r) != "" }

func denialKind(r *dns.Msg) string {
	for _, rr := range r.Ns {
		switch rr.Header().Rrtype {
		case dns.TypeNSEC3:
			return "NSEC3"
		case dns.TypeNSEC:
			return "NSEC"
		}
	}
	return ""
}
