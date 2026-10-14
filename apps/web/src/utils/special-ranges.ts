import type { Cidr, IPAddress } from './ip'
import { cidrContains, parseCidr } from './ip'

/**
 * IANA IPv4/IPv6 Special-Purpose Address Registries (plus the multicast blocks), as of 2026.
 * https://www.iana.org/assignments/iana-ipv4-special-registry
 * https://www.iana.org/assignments/iana-ipv6-special-registry
 */
export type RangeKind
  = | 'this-network' | 'private' | 'shared' | 'loopback' | 'link-local' | 'protocol' | 'documentation'
    | 'benchmarking' | 'reserved' | 'broadcast' | 'multicast' | 'translation' | 'discard' | 'tunnel'
    | 'unique-local' | 'anycast' | 'deprecated' | 'unspecified' | 'sr-sid'

export interface SpecialRange {
  cidr: string
  name: string
  rfc: string
  kind: RangeKind
  globallyReachable: boolean
}

function R(cidr: string, name: string, rfc: string, kind: RangeKind, globallyReachable = false): SpecialRange {
  return { cidr, name, rfc, kind, globallyReachable }
}

export const SPECIAL_RANGES: SpecialRange[] = [
  // IPv4
  R('0.0.0.0/8', '"This network"', 'RFC 791', 'this-network'),
  R('0.0.0.0/32', '"This host on this network"', 'RFC 1122', 'unspecified'),
  R('10.0.0.0/8', 'Private-Use', 'RFC 1918', 'private'),
  R('100.64.0.0/10', 'Shared Address Space (CGNAT)', 'RFC 6598', 'shared'),
  R('127.0.0.0/8', 'Loopback', 'RFC 1122', 'loopback'),
  R('169.254.0.0/16', 'Link Local', 'RFC 3927', 'link-local'),
  R('172.16.0.0/12', 'Private-Use', 'RFC 1918', 'private'),
  R('192.0.0.0/24', 'IETF Protocol Assignments', 'RFC 6890', 'protocol'),
  R('192.0.0.0/29', 'IPv4 Service Continuity Prefix', 'RFC 7335', 'protocol'),
  R('192.0.0.8/32', 'IPv4 dummy address', 'RFC 7600', 'protocol'),
  R('192.0.0.9/32', 'Port Control Protocol Anycast', 'RFC 7723', 'anycast', true),
  R('192.0.0.10/32', 'Traversal Using Relays around NAT Anycast', 'RFC 8155', 'anycast', true),
  R('192.0.0.170/32', 'NAT64/DNS64 Discovery', 'RFC 8880', 'translation'),
  R('192.0.0.171/32', 'NAT64/DNS64 Discovery', 'RFC 8880', 'translation'),
  R('192.0.2.0/24', 'Documentation (TEST-NET-1)', 'RFC 5737', 'documentation'),
  R('192.31.196.0/24', 'AS112-v4', 'RFC 7535', 'anycast', true),
  R('192.52.193.0/24', 'AMT', 'RFC 7450', 'tunnel', true),
  R('192.88.99.0/24', 'Deprecated (6to4 Relay Anycast)', 'RFC 7526', 'deprecated'),
  R('192.168.0.0/16', 'Private-Use', 'RFC 1918', 'private'),
  R('192.175.48.0/24', 'Direct Delegation AS112 Service', 'RFC 7534', 'anycast', true),
  R('198.18.0.0/15', 'Benchmarking', 'RFC 2544', 'benchmarking'),
  R('198.51.100.0/24', 'Documentation (TEST-NET-2)', 'RFC 5737', 'documentation'),
  R('203.0.113.0/24', 'Documentation (TEST-NET-3)', 'RFC 5737', 'documentation'),
  R('224.0.0.0/4', 'Multicast', 'RFC 5771', 'multicast'),
  R('240.0.0.0/4', 'Reserved', 'RFC 1112', 'reserved'),
  R('255.255.255.255/32', 'Limited Broadcast', 'RFC 919', 'broadcast'),
  // IPv6
  R('::/128', 'Unspecified Address', 'RFC 4291', 'unspecified'),
  R('::1/128', 'Loopback Address', 'RFC 4291', 'loopback'),
  R('::ffff:0:0/96', 'IPv4-mapped Address', 'RFC 4291', 'translation'),
  R('64:ff9b::/96', 'IPv4-IPv6 Translation (NAT64)', 'RFC 6052', 'translation', true),
  R('64:ff9b:1::/48', 'IPv4-IPv6 Translation (local use)', 'RFC 8215', 'translation'),
  R('100::/64', 'Discard-Only Address Block', 'RFC 6666', 'discard'),
  R('100:0:0:1::/64', 'Dummy IPv6 Prefix', 'RFC 9780', 'protocol'),
  R('2001::/23', 'IETF Protocol Assignments', 'RFC 2928', 'protocol'),
  R('2001::/32', 'TEREDO', 'RFC 4380', 'tunnel', true),
  R('2001:1::1/128', 'Port Control Protocol Anycast', 'RFC 7723', 'anycast', true),
  R('2001:1::2/128', 'Traversal Using Relays around NAT Anycast', 'RFC 8155', 'anycast', true),
  R('2001:1::3/128', 'DNS-SD Service Registration Protocol Anycast', 'RFC 9665', 'anycast', true),
  R('2001:2::/48', 'Benchmarking', 'RFC 5180', 'benchmarking'),
  R('2001:3::/32', 'AMT', 'RFC 7450', 'tunnel', true),
  R('2001:4:112::/48', 'AS112-v6', 'RFC 7535', 'anycast', true),
  R('2001:10::/28', 'Deprecated (previously ORCHID)', 'RFC 4843', 'deprecated'),
  R('2001:20::/28', 'ORCHIDv2', 'RFC 7343', 'protocol', true),
  R('2001:30::/28', 'Drone Remote ID Protocol Entity Tags (DETs)', 'RFC 9374', 'protocol', true),
  R('2001:db8::/32', 'Documentation', 'RFC 3849', 'documentation'),
  R('2002::/16', '6to4', 'RFC 3056', 'tunnel', true),
  R('2620:4f:8000::/48', 'Direct Delegation AS112 Service', 'RFC 7534', 'anycast', true),
  R('3fff::/20', 'Documentation', 'RFC 9637', 'documentation'),
  R('5f00::/16', 'Segment Routing (SRv6) SIDs', 'RFC 9602', 'sr-sid'),
  R('fc00::/7', 'Unique-Local', 'RFC 4193', 'unique-local'),
  R('fe80::/10', 'Link-Local Unicast', 'RFC 4291', 'link-local'),
  R('ff00::/8', 'Multicast', 'RFC 4291', 'multicast'),
]

const PARSED: (SpecialRange & { parsed: Cidr })[] = SPECIAL_RANGES.map(r => ({ ...r, parsed: parseCidr(r.cidr)! }))

export interface Classification {
  /** All matching ranges, most specific first. */
  matches: SpecialRange[]
  /** Most specific match, if any. */
  primary?: SpecialRange
  /** Not globally routable on the public Internet (bogon). */
  bogon: boolean
  /** For IPv6: outside 2000::/3 global unicast and not otherwise special. */
  unallocated: boolean
}

const GLOBAL_UNICAST_V6 = parseCidr('2000::/3')!

export function classify(addr: IPAddress): Classification {
  const matches = PARSED
    .filter(r => cidrContains(r.parsed, addr))
    .sort((a, b) => b.parsed.prefix - a.parsed.prefix)
    .map(({ parsed: _p, ...r }) => r)
  const primary = matches[0]
  const unallocated = addr.version === 6 && !matches.length && !cidrContains(GLOBAL_UNICAST_V6, addr)
  return { matches, primary, bogon: unallocated || (primary ? !primary.globallyReachable : false), unallocated }
}
