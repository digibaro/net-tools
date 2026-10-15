# Changelog

All notable changes to NET-TOOLS. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project uses [Semantic Versioning](https://semver.org/). Release notes with install
instructions are on the [GitHub releases page](https://github.com/digibaro/net-tools-docker/releases).

## [Unreleased]

### Added
- 26 new tools (151 in total):
  - IP: IPv6 address analyzer, IP address plan (IPAM sheet), NAT64 / DNS64 check, country IP blocks.
  - DNS: open resolver test, reverse DNS coverage (FCrDNS), HTTPS / SVCB & ECH viewer, EDNS
    compliance test.
  - HTTP: CDN & WAF detector, cache header analyzer, mixed content checker, CSP builder & evaluator.
  - TLS: DANE / TLSA checker, certificate chain validator, PKCS#12 / PFX inspector, lab CA &
    certificate generator.
  - Email: SMTP banner & FCrDNS check.
  - BGP: route history & visibility, ASN neighbour graph, ROA coverage per ASN, common IXPs &
    facilities.
  - Calculators: SLA & uptime, VoIP bandwidth & Erlang B, packet decoder, pcap viewer.
  - Client: DNS hijack & proxy check.
- HTTP/2 & HTTP/3 check: a real HTTP/3 request, TTFB compared with TCP, session resumption and
  0-RTT.
- Traceroute and MTR: AS path and the route on a world map (with geolocation available).

## [0.5.0] - 2026-10-15

### Added
- LAN tools: **LLDP / CDP neighbours** (which switch and port the server is plugged into),
  **IPv6 neighbours & routers** (NDP scan plus router advertisements: prefixes, SLAAC / DHCPv6,
  DNS, MTU) and **DHCPv6 probe** (rogue DHCPv6 check; never takes a lease). 125 tools in total.
- Dutch interface (Nederlands) with an NL / UK switch in the top bar; the choice is remembered and
  sets the page language. Numbers follow the chosen language (decimal comma in Dutch). A unit test
  keeps every translation in step with English.
- World map of probe locations for every Worldwide (Globalping) result; select a dot to jump to it.
  The map is bundled (Natural Earth), no external tiles.
- CSV download for every result table, and a print / save-as-PDF button on every tool.
- Cloud / CDN IP lookup covers Microsoft Azure (Service Tags, with service and region).
- Optional audit log (`AUDIT_LOG=true`): one line per server-tool request, client IP hashed with
  `LOG_HASH_IPS`, request bodies never logged.
- Subtitle "Tools for Networkers" under the name in the sidebar.
- Favicon, sidebar logo and loading screen follow the accent colour.
- Screenshots in the README (light, dark, phone, worldwide map).
- This changelog.

### Changed
- The "New" badge marks only tools added after the latest release; the release date comes from
  the git tag at build time.
- The sidebar footer shows version and licence only.
- On phones the search box in the top bar becomes a search button, so every header button fits
  at 360 px.
- `make up` and related targets stamp local builds with the git version (for example
  `0.4.0-3-gabc1234`) instead of `dev`.
- Repository renamed to `digibaro/net-tools-docker`; Go module path follows.

### Fixed
- Choice buttons (DNS lookup "Run from", HTTP status classes, URL parser, WireGuard routing) ran
  off the screen on phones; they now wrap.

## [0.4.0] - 2026-10-14

First public release: 122 tools in 11 categories, published as multi-arch images
`ghcr.io/digibaro/nettools-web` and `ghcr.io/digibaro/nettools-api`.

### Added
- LAN mode tools: Wake-on-LAN, ARP scan, mDNS / DNS-SD browser, SSDP / UPnP discovery, DHCP probe
  (rogue DHCP) and gateway / NAT info (NAT-PMP, PCP, UPnP IGD, double-NAT detection).
- Server tools: port scanner, iperf3 bandwidth test, SNMP v2c/v3 get and walk, AS-SET / IRR
  expansion (bgpq4), open relay test, URL / domain reputation (Safe Browsing, VirusTotal,
  urlscan.io).
- Browser tools: router config snippets, AS path visualizer, DMARC aggregate report viewer,
  JWK / PEM / SSH key converter.
- Accent colours (green, blue, indigo, purple, orange, red), checked for WCAG AA contrast.
- Sidebar footer with version and licence; product name shown as NET-TOOLS.
- Release workflow publishing images with SBOM and provenance; gitleaks secret scan in CI.

### Fixed
- LAN override: the web container could not reach the api (every API call failed).
- Speed test: uploads were refused by nginx (413) and long tests hit the rate limit (429).
- TLS scanner: data race on the cipher map.
- Comment-only values in a copied `.env.example` broke startup or enabled tools with bogus keys.
- iperf profile server failed every test (read-only `/tmp`).

### Earlier phases
Phases 0–3 (foundations, 32 browser tools, DNS / connectivity / HTTP / TLS / email / BGP server
tools, providers and Worldwide measurements) were developed before the first release; see the git
history and `docs/IMPLEMENTATION_PLAN.md`.

[Unreleased]: https://github.com/digibaro/net-tools-docker/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/digibaro/net-tools-docker/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/digibaro/net-tools-docker/releases/tag/v0.4.0
