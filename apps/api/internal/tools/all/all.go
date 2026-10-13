// Package all lists every server tool; main mounts them and /capabilities reports them.
package all

import (
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnscaa"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsconsistency"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsencrypted"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnspropagation"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsreverse"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnstrace"
	"github.com/digibaro/net-tools/apps/api/internal/tools/mtr"
	"github.com/digibaro/net-tools/apps/api/internal/tools/ntpcheck"
	"github.com/digibaro/net-tools/apps/api/internal/tools/ping"
	"github.com/digibaro/net-tools/apps/api/internal/tools/portcheck"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tcping"
)

// Tools returns every registered tool, in catalog order.
func Tools() []tools.Tool {
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
	}
}
