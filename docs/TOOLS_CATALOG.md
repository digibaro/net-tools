# Tools Catalog — NetTools

The backlog of every tool. Claude Code updates the **Status** column (`⬜` todo, `🚧` in progress,
`✅` done, `⏸` deferred).

**Run mode**
- **B** Browser — pure client-side, works offline.
- **S** Server — runs in the `api` container (server's vantage point).
- **P** Public API — called by the `api` container (keys stay server-side); never from the browser
  unless marked *(browser-direct OK)*.
- **H** Hybrid — local by default, optional public-API mode (e.g. "Worldwide").
- **L** Requires `lan` profile (host networking).

**Phase**: 1 = MVP shell + browser tools, 2 = core server tools, 3 = public APIs, 4 = advanced/LAN.

> Public-API notes reflect what was known when this kit was written. Before implementing a provider,
> check its current docs, terms and rate limits; mark keyed ones with `requiresKey` in capabilities.

---

## 1. IP & Addressing

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| IP-01 | What is my IP | Shows client public IPv4/IPv6 as seen by server + by outside | H | `api` echoes `X-Forwarded-For`/remote addr | ipify (`api.ipify.org`, `api6.ipify.org`) *(browser-direct OK)* | 1 | ✅ |
| IP-02 | IPv4 subnet calculator | Network, broadcast, mask, wildcard, host range, host count, class, private/public | B | `ipaddr.js` | — | 1 | ✅ |
| IP-03 | IPv6 subnet calculator | Prefix, range, number of /64s, compressed & expanded forms | B | `ip-address` + BigInt | — | 1 | ✅ |
| IP-04 | CIDR ↔ IP range | Convert CIDR to first/last IP and an arbitrary range to minimal CIDR list | B | custom | — | 1 | ✅ |
| IP-05 | CIDR aggregator / summarizer | Merge/deduplicate a list of prefixes (v4+v6) | B | custom (sorted merge) | — | 1 | ✅ |
| IP-06 | VLSM planner | Split a block into subnets by required host counts | B | custom | — | 2 | ✅ |
| IP-07 | Subnet splitter | Divide a prefix into N equal subnets / by new prefix length | B | custom | — | 1 | ✅ |
| IP-08 | IP in subnet checker | Is IP X inside any of the prefixes in a list? | B | `ipaddr.js` | — | 1 | ✅ |
| IP-09 | IPv4 format converter | Dotted ↔ decimal ↔ hex ↔ binary ↔ octal | B | custom | — | 1 | ✅ |
| IP-10 | IPv6 expand / compress | RFC 5952 canonical form, full form, reverse nibble | B | `ip-address` | — | 1 | ✅ |
| IP-11 | IPv6 ULA generator | RFC 4193 random /48 Unique Local prefix | B | WebCrypto | — | 1 | ✅ |
| IP-12 | IPv4-mapped / 6to4 / NAT64 | Embed/extract IPv4 in IPv6 (::ffff:, 2002::, 64:ff9b::/96) | B | custom | — | 2 | ✅ |
| IP-13 | Special-purpose / bogon check | Identify RFC 1918, CGNAT, loopback, doc, multicast, reserved ranges | B | IANA special registry (bundled JSON) | — | 1 | ✅ |
| IP-14 | Wildcard / ACL mask calculator | Subnet mask ↔ wildcard (Cisco ACL) | B | custom | — | 1 | ✅ |
| IP-15 | Reverse DNS name builder | IP → `in-addr.arpa` / `ip6.arpa` name | B | custom | — | 1 | ✅ |
| IP-16 | MAC address lookup (OUI) | Vendor from MAC prefix | B | `oui-data` bundled | macvendors.com API | 1 | ✅ |
| IP-17 | MAC formatter / generator | Normalize formats (`:`, `-`, `.`), random locally-administered MAC | B | custom | — | 1 | ✅ |
| IP-18 | EUI-64 / link-local from MAC | Build IPv6 interface ID / fe80:: address from MAC | B | custom | — | 1 | ✅ |
| IP-19 | Random IP generator | Random IPs within a prefix (test data) | B | custom | — | 2 | ✅ |

## 2. DNS

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| DNS-01 | DNS lookup | Query any record type against chosen resolver (system, 1.1.1.1, 8.8.8.8, 9.9.9.9, custom) | H | `miekg/dns` (UDP/TCP) | Cloudflare / Google DoH JSON *(browser-direct OK)* | 2 | ✅ |
| DNS-02 | Reverse DNS (PTR) | IP → hostname, with forward-confirmed check | S | `miekg/dns` | DoH | 2 | ✅ |
| DNS-03 | DNS propagation check | Same query across ~20 public resolvers, diff answers | H | `miekg/dns` fan-out | Globalping `dns` from many countries | 2 | ✅ |
| DNS-04 | Delegation trace | Iterative resolution from root (`dig +trace` equivalent) | S | `miekg/dns` iterative | — | 2 | ✅ |
| DNS-05 | DNSSEC validator | Validate chain (DS/DNSKEY/RRSIG), show where it breaks | S | `miekg/dns` + validation | — | 3 | ⬜ |
| DNS-06 | DoH / DoT tester | Query a DoH/DoT endpoint, show latency and response | S | `miekg/dns` + `net/http` | — | 2 | ✅ |
| DNS-07 | Resolver benchmark | Latency/success of N resolvers over M queries | S | fan-out | — | 3 | ⬜ |
| DNS-08 | Zone transfer (AXFR) test | Check if NS servers allow AXFR (security audit) | S | `miekg/dns` | — | 3 | ⬜ |
| DNS-09 | SOA / NS consistency | Compare SOA serials and NS sets across all authoritative servers | S | `miekg/dns` | — | 2 | ✅ |
| DNS-10 | CAA checker | CAA records with tree climbing per RFC 8659 | S | `miekg/dns` | — | 2 | ✅ |
| DNS-11 | Wildcard detection | Detect wildcard records via random labels | S | `miekg/dns` | — | 3 | ⬜ |
| DNS-12 | Zone file / record builder | Generate BIND-style records (SRV, TXT splitting, SPF) | B | custom | — | 2 | ✅ |
| DNS-13 | Punycode / IDN converter | Unicode domain ↔ `xn--` | B | `punycode` / `URL` | — | 1 | ✅ |
| DNS-14 | Domain parser | Split subdomain / registrable domain / public suffix | B | `tldts` | — | 1 | ✅ |

## 3. Connectivity & Diagnostics

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| CON-01 | Ping (ICMP) | Live RTT, loss, jitter, min/avg/max, chart | H | `pro-bing` (unprivileged via `ping_group_range` or NET_RAW) | Globalping `ping` | 2 | ✅ |
| CON-02 | TCP ping | Connect-time to host:port (ICMP-blocked targets) | S | `net.Dialer` loop | — | 2 | ✅ |
| CON-03 | Traceroute | ICMP / UDP / TCP hop list with RTTs, ASN & geo per hop | H | `mtr --report --json` or `traceroute` via execx | Globalping `traceroute` | 2 | ✅ |
| CON-04 | MTR | Continuous traceroute+ping, per-hop loss/latency table | H | `mtr --raw` streamed | Globalping `mtr` | 2 | ✅ |
| CON-05 | Port check | Is TCP (or UDP best-effort) port open from the server | S | `net.Dialer` | check-host.net API | 2 | ✅ |
| CON-06 | Port scanner | Bounded TCP connect scan (≤ `MAX_PORTS`), common-ports presets | S | Go worker pool; optional `nmap -sT` | Shodan InternetDB (passive) | 4 | ⬜ |
| CON-07 | Path MTU discovery | Find max non-fragmented packet size to target | S | `pro-bing` with DF + size bisection | — | 3 | ⬜ |
| CON-08 | Bandwidth test (iperf3) | TCP/UDP throughput to an iperf3 server | S | `iperf3 -c … --json` via execx | public iperf3 servers list (user-supplied) | 4 | ⬜ |
| CON-09 | Browser speed test | Download/upload/latency between browser and server | B+S | LibreSpeed container (`speedtest` profile) | Cloudflare speed test endpoints | 3 | ⬜ |
| CON-10 | Global ping / "is it down" | Ping/HTTP from many countries, map view | P | — | Globalping API, check-host.net | 3 | ⬜ |
| CON-11 | NTP check | Query NTP server: offset, stratum, delay, refid | S | `github.com/beevik/ntp` | — | 2 | ✅ |
| CON-12 | SNMP get / walk | v2c/v3 get/walk of an OID subtree (bounded) | S | `gosnmp` | — | 4 | ⬜ |
| CON-13 | Server network info | Interfaces, addresses, routes, resolvers, gateway of the api container/host | S | `net.Interfaces`, `/proc/net/route` | — | 2 | ✅ |
| CON-14 | IPv6 readiness | Does target have AAAA, is it reachable over v6 from server | S | DNS + dial | Globalping | 3 | ⬜ |
| CON-15 | Latency matrix | Ping multiple targets in parallel, compare | S | `pro-bing` fan-out | Globalping | 3 | ⬜ |

## 4. HTTP & Web

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| WEB-01 | HTTP request inspector | Method, headers, status, response headers, body preview | S | `net/http` w/ pinned dialer | Globalping `http` | 2 | ✅ |
| WEB-02 | Redirect chain tracer | Every hop with status, Location, timing | S | `net/http` no-follow loop | — | 2 | ✅ |
| WEB-03 | Timing breakdown | DNS / connect / TLS / TTFB / transfer (curl `-w` style) | S | `httptrace` | — | 2 | ✅ |
| WEB-04 | Security headers analyzer | CSP, HSTS, XFO, XCTO, Referrer-Policy, Permissions-Policy grading | S | custom rules | Mozilla HTTP Observatory API | 2 | ✅ |
| WEB-05 | HTTP/2 & HTTP/3 support | ALPN negotiation, `Alt-Svc`, QUIC handshake | S | `crypto/tls` ALPN + `quic-go` | — | 3 | ⬜ |
| WEB-06 | CORS tester | Send preflight with given Origin/method/headers, evaluate | S | `net/http` | — | 2 | ✅ |
| WEB-07 | Compression check | gzip / br / zstd support and ratio | S | `net/http` | — | 3 | ⬜ |
| WEB-08 | Cookie inspector | Parse Set-Cookie flags (Secure, HttpOnly, SameSite) | S | `net/http` | — | 3 | ⬜ |
| WEB-09 | robots.txt / sitemap / security.txt | Fetch & parse well-known files | S | `net/http` | — | 3 | ⬜ |
| WEB-10 | HSTS preload status | Is domain on the Chromium preload list | P | — | hstspreload.org API | 3 | ⬜ |
| WEB-11 | WebSocket tester | Connect, send/receive messages, close codes | B | native `WebSocket` | — | 3 | ⬜ |
| WEB-12 | URL parser / encoder | Components, query params, encode/decode | B | `URL` | — | 1 | ✅ |
| WEB-13 | User-agent parser | Browser/OS/device from UA string | B | `ua-parser-js` | — | 1 | ✅ |
| WEB-14 | HTTP status code reference | Searchable list with RFC links | B | bundled JSON | — | 1 | ✅ |
| WEB-15 | curl command builder / parser | Build curl from form; parse curl into fields | B | custom | — | 3 | ⬜ |

## 5. TLS & Certificates

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| TLS-01 | TLS certificate inspector | Fetch chain from host:port (SNI), show fields, SANs, validity, issuer chain | S | `crypto/tls` + `x509` | — | 2 | ✅ |
| TLS-02 | Certificate expiry checker | Days left for one or many hosts | S | `crypto/tls` | — | 2 | ✅ |
| TLS-03 | TLS protocol & cipher scan | Supported TLS 1.0–1.3, cipher suites, weak configs | S | Go handshake loop (or `testssl.sh` opt-in image) | SSL Labs API v4 (registration required) | 3 | ⬜ |
| TLS-04 | OCSP / CRL status | Revocation status of leaf cert | S | `x/crypto/ocsp` | — | 3 | ⬜ |
| TLS-05 | Certificate decoder | Paste PEM/DER → human-readable fields | B | `@peculiar/x509` | — | 1 | ✅ |
| TLS-06 | CSR decoder / generator | Decode CSR; generate key+CSR in browser | B | `@peculiar/x509` + WebCrypto | — | 2 | ✅ |
| TLS-07 | Cert / key matcher | Check that a cert, key and CSR belong together | B | WebCrypto | — | 2 | ⬜ |
| TLS-08 | Certificate Transparency search | Certificates issued for a domain (subdomain discovery) | P | — | crt.sh JSON (`?output=json`) | 3 | ⬜ |
| TLS-09 | STARTTLS check | SMTP/IMAP/POP3/FTP/XMPP STARTTLS + cert | S | `net/smtp` & raw dialogs | — | 3 | ⬜ |
| TLS-10 | SSH server audit | Banner, host key fingerprints, KEX/cipher algorithms | S | `x/crypto/ssh` handshake (no auth) | — | 3 | ⬜ |
| TLS-11 | JWK / PEM converter | Convert keys between PEM, JWK, SSH formats | B | WebCrypto | — | 4 | ⬜ |

## 6. Email

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| EM-01 | MX lookup + SMTP reachability | MX records, connect each on 25, banner, STARTTLS | S | `miekg/dns` + `net/smtp` | — | 2 | ✅ |
| EM-02 | SPF checker | Parse, expand includes, count DNS lookups (≤10), flag errors | S | custom on `miekg/dns` | — | 2 | ✅ |
| EM-03 | DMARC checker | Parse policy, rua/ruf, alignment | S | custom | — | 2 | ✅ |
| EM-04 | DKIM checker | Fetch `<selector>._domainkey`, key size/type | S | custom | — | 2 | ✅ |
| EM-05 | MTA-STS / TLS-RPT / BIMI | Fetch policy file + DNS, validate | S | custom | — | 3 | ⬜ |
| EM-06 | Blacklist (DNSBL) check | Query IP/domain against ~30 RBLs | S | DNS queries | — | 3 | ⬜ |
| EM-07 | Email header analyzer | Paste headers → hop timeline, delays, auth results | B | custom parser | — | 2 | ✅ |
| EM-08 | DMARC aggregate report viewer | Upload XML/zip/gz report → tables | B | `fflate` + XML parser | — | 4 | ⬜ |
| EM-09 | Open relay test | Opt-in, bounded relay test of own server | S | `net/smtp` (no DATA sent) | — | 4 | ⬜ |

## 7. Routing, BGP & Registry

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| BGP-01 | IP → ASN | Origin ASN, prefix, AS name, RIR | H | Team Cymru DNS (`origin.asn.cymru.com`) via `miekg/dns` | RIPEstat `prefix-overview` | 2 | ✅ |
| BGP-02 | ASN info | Name, country, announced prefixes v4/v6, neighbours | P | — | RIPEstat `as-overview`, `announced-prefixes`, `asn-neighbours` | 3 | ⬜ |
| BGP-03 | Prefix routing status | Visibility, first/last seen, origin(s), MOAS detection | P | — | RIPEstat `routing-status`, `bgp-state` | 3 | ⬜ |
| BGP-04 | RPKI ROV validation | Valid/invalid/unknown for prefix+origin | P | — | RIPEstat `rpki-validation` | 3 | ⬜ |
| BGP-05 | Looking glass | Routes seen by RIS collectors | P | — | RIPEstat `looking-glass` | 3 | ⬜ |
| BGP-06 | WHOIS | Domain / IP / ASN WHOIS (port 43, referral following) | S | own port-43 client (every referral via `target.Resolve`) | — | 2 | ✅ |
| BGP-07 | RDAP lookup | Structured registration data (domain, IP, ASN) | P | `openrdap` client | rdap.org bootstrap / IANA bootstrap | 2 | ✅ |
| BGP-08 | Domain expiry checker | Registration and expiry dates via RDAP | P | `openrdap` | RDAP | 2 | ✅ |
| BGP-09 | PeeringDB lookup | Networks, IXPs, facilities, peering policy for ASN | P | — | PeeringDB API (optional key) | 3 | ⬜ |
| BGP-10 | AS-SET / IRR expansion | Expand AS-SET to ASNs/prefixes (filter building) | S | `bgpq4` via execx | — | 4 | ⬜ |
| BGP-11 | AS path / prepend visualizer | Paste AS path(s) → graph | B | custom (echarts graph) | — | 4 | ⬜ |
| BGP-12 | Cloud / CDN IP range lookup | Which provider/region owns this IP (AWS, GCP, Azure, Cloudflare, Fastly…) | S | cached published JSON lists refreshed daily | — | 3 | ⬜ |

## 8. Reputation & Geolocation

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| REP-01 | IP geolocation | Country/city/ASN/ISP | H | MaxMind GeoLite2 / DB-IP Lite mmdb mounted in `/data` | ipinfo.io (token), ip-api.com (free non-commercial) | 2 | ✅ |
| REP-02 | IP reputation | Abuse score, reports | P | — | AbuseIPDB (key) | 3 | ⬜ |
| REP-03 | Passive exposure | Open ports, CPEs, known vulns (passive, no scanning) | P | — | Shodan InternetDB (keyless) | 3 | ⬜ |
| REP-04 | Tor exit node check | Is IP a current Tor exit | S | cached Tor bulk exit list | Tor Onionoo | 3 | ⬜ |
| REP-05 | URL / domain reputation | Malware/phishing verdicts | P | — | Google Safe Browsing (key), VirusTotal (key), urlscan.io (key) | 4 | ⬜ |

## 9. LAN (requires `lan` profile)

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| LAN-01 | Wake-on-LAN | Send magic packet to MAC (broadcast / directed) | S L | UDP broadcast | — | 4 | ⬜ |
| LAN-02 | ARP scan / host discovery | Live hosts in local subnet with MAC + vendor | S L | `mdlayher/arp` (needs NET_RAW) or `arp-scan` | — | 4 | ⬜ |
| LAN-03 | mDNS / Bonjour browser | Discover `_services._dns-sd._udp` | S L | `hashicorp/mdns` / `grandcat/zeroconf` | — | 4 | ⬜ |
| LAN-04 | SSDP / UPnP discovery | Discover UPnP devices, read descriptions | S L | UDP multicast M-SEARCH | — | 4 | ⬜ |
| LAN-05 | DHCP probe | Detect DHCP servers (rogue DHCP check) | S L | `insomniacslk/dhcp` (needs NET_RAW/ADMIN) | — | 4 | ⬜ |
| LAN-06 | Gateway / NAT info | Default gateway, NAT-PMP/PCP/UPnP external IP | S L | custom | — | 4 | ⬜ |

## 10. Calculators & Reference

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| CAL-01 | Bandwidth / transfer-time calculator | Size ÷ speed, with protocol overhead | B | custom | — | 1 | ✅ |
| CAL-02 | Bits/bytes unit converter | bit, B, Kb/KiB, Mbps ↔ MB/s, … | B | custom | — | 1 | ✅ |
| CAL-03 | BDP / TCP window calculator | Bandwidth-delay product, required window, max throughput for given window+RTT | B | custom | — | 1 | ✅ |
| CAL-04 | MTU / MSS calculator | Overheads for VLAN, QinQ, PPPoE, GRE, IPsec, WireGuard, VXLAN, GENEVE | B | custom table | — | 1 | ✅ |
| CAL-05 | Optical power converter | dBm ↔ mW, link budget | B | custom | — | 2 | ✅ |
| CAL-06 | Port number reference | IANA service names/ports, searchable | B | bundled IANA CSV → JSON | — | 1 | ✅ |
| CAL-07 | Protocol / ICMP type reference | IP protocol numbers, ICMP/ICMPv6 types & codes | B | bundled JSON | — | 1 | ✅ |
| CAL-08 | DNS record type reference | Record types, RFCs, examples | B | bundled JSON | — | 1 | ✅ |
| CAL-09 | Subnet cheat sheet | /0–/32 and common v6 prefix table | B | generated | — | 1 | ✅ |
| CAL-10 | WireGuard config generator | Keypairs (x25519), server/peer configs, QR | B | `@noble/curves` + `qrcode` | — | 2 | ⬜ |
| CAL-11 | Wi-Fi QR generator | WIFI: URI → QR | B | `qrcode` | — | 1 | ✅ |
| CAL-12 | Cisco / Juniper snippet helpers | Prefix-list / ACL from CIDR list | B | templates | — | 4 | ⬜ |

## 11. Client Diagnostics (runs in the user's browser)

| ID | Tool | What it does | Run | Local implementation | Public API option | Phase | Status |
|---|---|---|---|---|---|---|---|
| CLI-01 | Browser ↔ server latency | HTTP RTT from browser to NetTools api, jitter graph | B+S | `fetch` to `/api/v1/echo` | — | 2 | ⬜ |
| CLI-02 | WebRTC IP leak test | Local/public IPs exposed via ICE candidates | B | `RTCPeerConnection` | STUN (stun.l.google.com, stun.cloudflare.com) | 3 | ⬜ |
| CLI-03 | Browser connection info | Network Information API, IPv4/IPv6 reachability, DoH in use? | B | `navigator.connection` + fetch to v4/v6-only endpoints | ipify v4/v6 endpoints | 2 | ⬜ |
| CLI-04 | Request headers echo | Headers the server receives from your browser (proxy detection) | S | `/api/v1/echo` | — | 1 | ✅ |

---

### Totals

| Category | Tools |
|---|---|
| IP & Addressing | 19 |
| DNS | 14 |
| Connectivity & Diagnostics | 15 |
| HTTP & Web | 15 |
| TLS & Certificates | 11 |
| Email | 9 |
| Routing, BGP & Registry | 12 |
| Reputation & Geolocation | 5 |
| LAN | 6 |
| Calculators & Reference | 12 |
| Client Diagnostics | 4 |
| **Total** | **122** |

### Public API providers summary

| Provider | Used by | Key | Notes |
|---|---|---|---|
| Globalping (jsDelivr) — `api.globalping.io/v1/measurements` | DNS-03, CON-01/03/04/10/14/15, WEB-01 | Optional token (higher limits) | ping, traceroute, mtr, dns, http from worldwide probes |
| RIPEstat Data API — `stat.ripe.net/data/<call>/data.json` | BGP-01..05 | No (add `sourceapp` param) | Prefix/ASN/routing/RPKI/looking glass |
| RDAP (rdap.org / IANA bootstrap) | BGP-07/08 | No | Successor of WHOIS for structured data |
| PeeringDB — `www.peeringdb.com/api` | BGP-09 | Optional | Anonymous rate limits are low; cache |
| crt.sh | TLS-08 | No | Can be slow; long timeout + cache |
| Cloudflare / Google DoH | DNS-01 | No | Browser-direct OK |
| ipify | IP-01, CLI-03 | No | Browser-direct OK |
| ipinfo.io / ip-api.com | REP-01 | ipinfo token; ip-api free tier non-commercial | Prefer local mmdb |
| Shodan InternetDB | REP-03, CON-06 | No | Passive data only |
| AbuseIPDB | REP-02 | Yes | |
| Mozilla HTTP Observatory | WEB-04 | No | |
| hstspreload.org | WEB-10 | No | |
| SSL Labs API v4 | TLS-03 | Registration (email) | Slow; async polling |
| check-host.net | CON-05, CON-10 | No | Alternative multi-location checks |
| macvendors.com | IP-16 | No (rate-limited) | Fallback only; bundled OUI preferred |
| Google Safe Browsing / VirusTotal / urlscan.io | REP-05 | Yes | Phase 4 |
