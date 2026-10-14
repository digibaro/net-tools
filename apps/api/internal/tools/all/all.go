// Package all lists every server tool; main mounts them and /capabilities reports them.
package all

import (
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/asnlookup"
	"github.com/digibaro/net-tools/apps/api/internal/tools/cloudranges"
	"github.com/digibaro/net-tools/apps/api/internal/tools/ctsearch"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsaxfr"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsbench"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsbl"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnscaa"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsconsistency"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsencrypted"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnspropagation"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsreverse"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnssec"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnstrace"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnswildcard"
	"github.com/digibaro/net-tools/apps/api/internal/tools/email"
	"github.com/digibaro/net-tools/apps/api/internal/tools/geo"
	"github.com/digibaro/net-tools/apps/api/internal/tools/hstspreload"
	"github.com/digibaro/net-tools/apps/api/internal/tools/ipv6ready"
	"github.com/digibaro/net-tools/apps/api/internal/tools/latmatrix"
	"github.com/digibaro/net-tools/apps/api/internal/tools/mtr"
	"github.com/digibaro/net-tools/apps/api/internal/tools/ntpcheck"
	"github.com/digibaro/net-tools/apps/api/internal/tools/observatory"
	"github.com/digibaro/net-tools/apps/api/internal/tools/peeringdb"
	"github.com/digibaro/net-tools/apps/api/internal/tools/ping"
	"github.com/digibaro/net-tools/apps/api/internal/tools/pmtu"
	"github.com/digibaro/net-tools/apps/api/internal/tools/portcheck"
	"github.com/digibaro/net-tools/apps/api/internal/tools/rdap"
	"github.com/digibaro/net-tools/apps/api/internal/tools/remote"
	"github.com/digibaro/net-tools/apps/api/internal/tools/reputation"
	"github.com/digibaro/net-tools/apps/api/internal/tools/revocation"
	"github.com/digibaro/net-tools/apps/api/internal/tools/ripestat"
	"github.com/digibaro/net-tools/apps/api/internal/tools/serverinfo"
	"github.com/digibaro/net-tools/apps/api/internal/tools/sshaudit"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tcping"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tlsinspect"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tlsscan"
	"github.com/digibaro/net-tools/apps/api/internal/tools/torcheck"
	"github.com/digibaro/net-tools/apps/api/internal/tools/web"
	"github.com/digibaro/net-tools/apps/api/internal/tools/whois"
)

// Tools returns every registered tool, in catalog order.
func Tools() []tools.Tool {
	rdapSvc := &rdap.Service{}
	return []tools.Tool{
		&dnslookup.Tool{},
		&dnsreverse.Tool{},
		&dnspropagation.Tool{},
		&dnstrace.Tool{},
		&dnsencrypted.Tool{},
		&dnsconsistency.Tool{},
		&dnscaa.Tool{},
		&ping.Tool{},
		&tcping.Tool{},
		&mtr.Tool{Mode: mtr.Traceroute},
		&mtr.Tool{Mode: mtr.MTR},
		&portcheck.Tool{},
		&ntpcheck.Tool{},
		&serverinfo.Tool{},
		&web.RequestTool{},
		&web.RedirectsTool{},
		&web.TimingTool{},
		&web.SecHeadersTool{},
		&web.CORSTool{},
		&tlsinspect.InspectTool{},
		&tlsinspect.ExpiryTool{},
		&email.MXTool{},
		&email.SPFTool{},
		&email.DMARCTool{},
		&email.DKIMTool{},
		&email.MTASTSTool{},
		&asnlookup.Tool{},
		&whois.Tool{},
		&rdap.LookupTool{Svc: rdapSvc},
		&rdap.ExpiryTool{Svc: rdapSvc},
		&geo.Tool{},
		&remote.Tool{},
		&ripestat.ASNTool{},
		&ripestat.PrefixTool{},
		&ripestat.RPKITool{},
		&ripestat.LookingGlassTool{},
		&peeringdb.Tool{},
		&ctsearch.Tool{},
		&reputation.InternetDBTool{},
		&reputation.AbuseIPDBTool{},
		&torcheck.Tool{},
		&hstspreload.Tool{},
		&cloudranges.Tool{},
		&observatory.Tool{},
		&dnsaxfr.Tool{},
		&dnswildcard.Tool{},
		&dnsbench.Tool{},
		&dnssec.Tool{},
		&web.CompressionTool{},
		&web.CookiesTool{},
		&web.WellKnownTool{},
		&web.ProtocolsTool{},
		&revocation.Tool{},
		&tlsscan.Tool{},
		&sshaudit.Tool{},
		&dnsbl.Tool{},
		&latmatrix.Tool{},
		&pmtu.Tool{},
		&ipv6ready.Tool{},
	}
}
