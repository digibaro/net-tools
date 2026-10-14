# API Specification — NetTools `api`

Base path: `/api` (proxied by nginx). Versioned routes under `/api/v1`. The source of truth for
shapes is `apps/api/openapi.yaml` (OpenAPI 3.1); frontend types are generated with
`openapi-typescript` (`pnpm gen:api`). This document defines conventions; every endpoint added must
follow them.

## 1. Conventions

- JSON only, UTF-8, `camelCase` fields.
- Request/response tools use `GET` with query params when idempotent and short (shareable URLs);
  `POST` with JSON body for large inputs (bulk lists).
- Streaming tools use **SSE** (`text/event-stream`) over `GET …/stream?…` so the browser can use
  `EventSource`. Closing the connection cancels the job.
- Every response carries `X-Request-Id`.
- Durations in milliseconds (`float64`, 3 decimals), timestamps RFC 3339 UTC.

### 1.1 Envelope

```json
{
  "ok": true,
  "data": { "...tool-specific..." },
  "meta": {
    "tool": "dns.lookup",
    "target": { "input": "example.com", "resolved": ["93.184.215.14"], "used": "93.184.215.14" },
    "vantage": "server",            // "server" | "globalping" | provider id
    "durationMs": 23.412,
    "cached": false
  }
}
```

Error:

```json
{ "ok": false, "error": { "code": "TARGET_PRIVATE_DENIED", "message": "…", "details": {} } }
```

### 1.2 Error codes

| HTTP | Code | Meaning |
|---|---|---|
| 400 | `INVALID_INPUT` | Validation failed (`details.field`) |
| 400 | `TARGET_UNRESOLVABLE` | DNS resolution failed |
| 403 | `TARGET_PRIVATE_DENIED` | Target is private/loopback/link-local and `ALLOW_PRIVATE_TARGETS=false` |
| 403 | `TARGET_BLOCKED` | Matches deny-list (metadata IPs, configured CIDRs) |
| 403 | `TOOL_DISABLED` | Feature flag off |
| 403 | `PROVIDER_KEY_MISSING` | Public provider requires a key not configured |
| 408 | `TIMEOUT` | Tool exceeded its timeout |
| 429 | `RATE_LIMITED` | Per-IP limit hit (`Retry-After` header set) |
| 503 | `BUSY` | Global concurrency pool for tool is full |
| 502 | `PROVIDER_ERROR` | Upstream public API failed (`details.status`) |
| 500 | `INTERNAL` | Bug; logged with request id |

## 2. System endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/healthz` | Liveness: `{"status":"ok"}` |
| GET | `/api/readyz` | Readiness: config loaded, mmdb loaded (if configured) |
| GET | `/api/v1/version` | `{version, commit, buildDate}` |
| GET | `/api/v1/capabilities` | Tool availability (below) |
| GET | `/api/v1/echo` | Client IP as seen, headers, protocol; used by IP-01, CLI-01, CLI-04 |
| GET | `/metrics` | Prometheus (only if `METRICS_ENABLED=true`, not proxied by nginx by default) |

### `/api/v1/capabilities`

```json
{
  "ok": true,
  "data": {
    "mode": { "allowPrivateTargets": false, "lan": false },
    "tools": {
      "dns.lookup":        { "enabled": true },
      "net.portscan":      { "enabled": false, "reason": "disabled_by_admin" },
      "rep.abuseipdb":     { "enabled": false, "reason": "provider_key_missing" },
      "lan.wol":           { "enabled": false, "reason": "requires_lan_mode" },
      "rep.geo":           { "enabled": false, "reason": "data_missing" }
    },
    "limits": { "pingMaxCount": 20, "tracerouteMaxHops": 30, "portscanMaxPorts": 1024 }
  }
}
```

Reasons: `disabled_by_admin`, `provider_key_missing`, `requires_lan_mode`, `binary_missing`, `data_missing` (a required data file such as the GeoIP mmdb is not mounted).

## 3. Tool endpoints (v1)

Capability id in brackets. `target` always accepts hostname, IPv4 or IPv6; optional `family=4|6`.

### DNS
| Method | Path | Params | [cap] |
|---|---|---|---|
| GET | `/api/v1/dns/lookup` | `name`, `type` (A,AAAA,CNAME,MX,NS,TXT,SOA,SRV,CAA,PTR,DS,DNSKEY,HTTPS,SVCB,TLSA,NAPTR,ANY), `resolver` (ip[:port] or preset id), `transport=udp|tcp|dot|doh`, `dnssec=bool` | dns.lookup |
| GET | `/api/v1/dns/reverse` | `ip` | dns.reverse |
| GET | `/api/v1/dns/propagation` | `name`, `type`, `resolvers` (preset set id) | dns.propagation |
| GET | `/api/v1/dns/trace/stream` | `name`, `type` → SSE `step` events | dns.trace |
| GET | `/api/v1/dns/dnssec` | `name` | dns.dnssec |
| GET | `/api/v1/dns/consistency` | `zone` | dns.consistency |
| GET | `/api/v1/dns/caa` | `name` | dns.caa |
| GET | `/api/v1/dns/axfr` | `zone` | dns.axfr |
| GET | `/api/v1/dns/benchmark/stream` | `resolvers`, `queries` | dns.benchmark |

### Connectivity
| Method | Path | Params | [cap] |
|---|---|---|---|
| GET | `/api/v1/ping/stream` | `target`, `count` (≤ `PING_MAX_COUNT`), `interval` (≥200ms), `size` | net.ping |
| GET | `/api/v1/tcping/stream` | `target`, `port`, `count` | net.tcping |
| GET | `/api/v1/traceroute/stream` | `target`, `proto=icmp|udp|tcp`, `port`, `maxHops` | net.traceroute |
| GET | `/api/v1/mtr/stream` | `target`, `proto`, `cycles` | net.mtr |
| GET | `/api/v1/port/check` | `target`, `ports` (≤ 10, comma list), `proto=tcp|udp` | net.portcheck |
| GET | `/api/v1/portscan/stream` | `target`, `ports` (range/preset, ≤ `PORTSCAN_MAX_PORTS`) | net.portscan |
| GET | `/api/v1/pmtu` | `target` | net.pmtu |
| GET | `/api/v1/iperf/stream` | `server`, `port`, `proto`, `duration` (≤ 30s), `reverse` | net.iperf |
| GET | `/api/v1/ntp` | `server` | net.ntp |
| POST | `/api/v1/snmp` | `{target, version, community|v3auth, oid, walk}` | net.snmp |
| GET | `/api/v1/server/netinfo` | — | net.serverinfo |

### HTTP / TLS / Email
| Method | Path | Params | [cap] |
|---|---|---|---|
| POST | `/api/v1/http/request` | `{url, method, headers, body?, followRedirects}` | web.request |
| GET | `/api/v1/http/redirects` | `url` | web.redirects |
| GET | `/api/v1/http/timing` | `url` | web.timing |
| GET | `/api/v1/http/security-headers` | `url` | web.secheaders |
| GET | `/api/v1/http/cors` | `url`, `origin`, `method`, `headers` | web.cors |
| GET | `/api/v1/http/protocols` | `url` (h2, h3, alt-svc) | web.protocols |
| GET | `/api/v1/tls/inspect` | `host`, `port=443`, `sni`, `starttls=smtp|imap|pop3|ftp|xmpp` | tls.inspect |
| POST | `/api/v1/tls/expiry` | `{targets: ["host:port", …]}` (≤ 50) | tls.expiry |
| GET | `/api/v1/tls/scan/stream` | `host`, `port` | tls.scan |
| GET | `/api/v1/ssh/audit` | `host`, `port=22` | ssh.audit |
| GET | `/api/v1/email/mx` | `domain` | email.mx |
| GET | `/api/v1/email/spf` | `domain` | email.spf |
| GET | `/api/v1/email/dmarc` | `domain` | email.dmarc |
| GET | `/api/v1/email/dkim` | `domain`, `selector` | email.dkim |
| GET | `/api/v1/email/mtasts` | `domain` | email.mtasts |
| GET | `/api/v1/email/dnsbl/stream` | `target` (ip or domain) | email.dnsbl |

### Registry / BGP / Reputation (mostly provider-backed)
| Method | Path | Params | [cap] |
|---|---|---|---|
| GET | `/api/v1/whois` | `query` | reg.whois |
| GET | `/api/v1/rdap` | `query`, `kind=domain|ip|asn` | reg.rdap |
| GET | `/api/v1/asn/lookup` | `ip` | bgp.ip2asn |
| GET | `/api/v1/asn/{asn}` | — | bgp.asn |
| GET | `/api/v1/prefix/status` | `prefix` | bgp.prefix |
| GET | `/api/v1/rpki` | `prefix`, `asn` | bgp.rpki |
| GET | `/api/v1/lookingglass` | `prefix` | bgp.lg |
| GET | `/api/v1/peeringdb/{asn}` | — | bgp.peeringdb |
| GET | `/api/v1/cloud-ranges` | `ip` | bgp.cloudranges |
| GET | `/api/v1/geo` | `ip`, `source=local|ipinfo|ipapi` | rep.geo |
| GET | `/api/v1/reputation/internetdb` | `ip` (passive exposure, keyless) | rep.internetdb |
| GET | `/api/v1/reputation/abuseipdb` | `ip` (needs `ABUSEIPDB_KEY`) | rep.abuseipdb |
| GET | `/api/v1/tor-exit` | `ip` | rep.tor |
| GET | `/api/v1/ct` | `domain`, `includeExpired` | tls.ct |

### Remote measurements (Globalping)
| Method | Path | Params | [cap] |
|---|---|---|---|
| GET | `/api/v1/remote/stream` | `type=ping|traceroute|mtr|dns|http`, `target`, `locations` (e.g. `Germany,US+AWS`), `limit` (≤ 10) | remote.globalping |

### LAN (lan mode only)
| Method | Path | Params | [cap] |
|---|---|---|---|
| POST | `/api/v1/lan/wol` | `{mac, broadcast?, port=9}` | lan.wol |
| GET | `/api/v1/lan/arpscan/stream` | `cidr` (must be a directly connected subnet, ≤ /22) | lan.arpscan |
| GET | `/api/v1/lan/mdns/stream` | `service?`, `seconds ≤ 10` | lan.mdns |
| GET | `/api/v1/lan/ssdp/stream` | `seconds ≤ 10` | lan.ssdp |
| GET | `/api/v1/lan/dhcp` | — | lan.dhcp |

## 4. SSE event schema

```
event: meta      data: {"tool":"net.traceroute","target":{…},"vantage":"server","params":{…}}
event: line      data: {"text":"raw output line"}                       # optional raw view
event: reply     data: {"seq":3,"rttMs":12.3,"ttl":56,"from":"1.2.3.4"}  # ping/tcping
event: hop       data: {"hop":4,"ip":"…","host":"…","asn":13335,"rttMs":[…],"loss":0}
event: port      data: {"port":443,"state":"open","service":"https","rttMs":4.1}
event: progress  data: {"done":120,"total":1024}
event: result    data: {…}                                               # single item for list tools
event: done      data: {"summary":{…},"durationMs":…}
event: error     data: {"code":"TIMEOUT","message":"…"}
: ping                                                                   # heartbeat every 15s
```

The frontend composable `useSse(url, handlers)` maps events to reactive state; `StreamConsole.vue`
renders `line` events, tool components render typed events. Local and Globalping streams use the same
`reply`/`hop` shapes (Globalping results get `probe: {country, city, asn, network}` added).

## 5. Caching

| Data | TTL |
|---|---|
| RIPEstat, PeeringDB, RDAP | 10 min |
| crt.sh | 30 min |
| Cloud IP ranges, Tor exit list | refreshed every 24 h / 1 h in background |
| GeoIP mmdb | file reload on change |
| DNS / ping / traceroute | never cached |
