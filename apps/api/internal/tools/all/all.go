// Package all lists every server tool; main mounts them and /capabilities reports them.
package all

import (
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnscaa"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsconsistency"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnspropagation"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnsreverse"
)

// Tools returns every registered tool, in catalog order.
func Tools() []tools.Tool {
	return []tools.Tool{
		&dnslookup.Tool{},
		&dnsreverse.Tool{},
		&dnspropagation.Tool{},
		&dnsconsistency.Tool{},
		&dnscaa.Tool{},
	}
}
