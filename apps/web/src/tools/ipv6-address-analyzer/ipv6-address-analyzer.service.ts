import type { Extracted } from '../ipv4-in-ipv6/ipv4-in-ipv6.service'
import { expandIPv6, formatIPv4, formatIPv6, parseIPv6 } from '@/utils/ip'
import { formatMac, isLocallyAdministered } from '@/utils/mac'
import { classify } from '@/utils/special-ranges'
import { extract } from '../ipv4-in-ipv6/ipv4-in-ipv6.service'
import { ip6Arpa } from '../ipv6-expand-compress/ipv6-expand-compress.service'

export type Scope = 'interface' | 'link' | 'realm' | 'admin' | 'site' | 'organization' | 'global' | 'host' | 'none'
/** How the interface identifier (low 64 bits) was probably formed. */
export type IidKind = 'eui64' | 'isatap' | 'low' | 'random' | 'zero'

export interface Ipv6Analysis {
  compressed: string
  expanded: string
  arpa: string
  binary: string
  range: string
  rangeName: string
  rfc: string
  globallyReachable: boolean
  multicast: boolean
  scope: Scope
  /** Multicast flags (RFC 4291 / 3306 / 3956): T = transient, P = prefix-based, R = embedded RP. */
  flags?: { transient: boolean, prefix: boolean, rp: boolean }
  /** Unicast only. */
  iid?: { kind: IidKind, value: string, mac?: string, macLocal?: boolean, ipv4?: string }
  prefix64?: string
  solicitedNode?: string
  embedded: Extracted[]
}

const MCAST_SCOPE: Record<number, Scope> = { 1: 'interface', 2: 'link', 3: 'realm', 4: 'admin', 5: 'site', 8: 'organization', 14: 'global' }
const LOW64 = (1n << 64n) - 1n

function iidOf(a: bigint): NonNullable<Ipv6Analysis['iid']> {
  const iid = a & LOW64
  const value = [48n, 32n, 16n, 0n].map(s => ((iid >> s) & 0xFFFFn).toString(16)).join(':')
  if (iid === 0n)
    return { kind: 'zero', value }
  if (((iid >> 24n) & 0xFFFFn) === 0xFFFEn) {
    const mac = (((iid >> 40n) ^ 0x020000n) << 24n) | (iid & 0xFFFFFFn)
    return { kind: 'eui64', value, mac: formatMac(mac), macLocal: isLocallyAdministered(mac) }
  }
  if (((iid >> 32n) & 0xFDFFFFFFn) === 0x5EFEn)
    return { kind: 'isatap', value, ipv4: formatIPv4(iid & 0xFFFFFFFFn) }
  if (iid < 0x10000n)
    return { kind: 'low', value }
  return { kind: 'random', value }
}

export function analyzeIPv6(input: string): Ipv6Analysis | undefined {
  const a = parseIPv6(input.trim().replace(/%.*$/, '').replace(/\/\d+$/, ''))
  if (a === undefined)
    return undefined
  const c = classify({ version: 6, value: a })
  const multicast = a >> 120n === 0xFFn
  const out: Ipv6Analysis = {
    compressed: formatIPv6(a),
    expanded: expandIPv6(a),
    arpa: ip6Arpa(a),
    binary: a.toString(2).padStart(128, '0').replace(/(.{16})(?!$)/g, '$1 '),
    range: c.primary?.cidr ?? '2000::/3',
    rangeName: c.primary?.name ?? (c.unallocated ? 'Unallocated / reserved by IETF' : 'Global Unicast'),
    rfc: c.primary?.rfc ?? '',
    globallyReachable: !c.bogon,
    multicast,
    scope: 'global',
    embedded: extract(formatIPv6(a)) ?? [],
  }
  if (multicast) {
    const f = Number((a >> 116n) & 0xFn)
    out.flags = { transient: (f & 1) === 1, prefix: (f & 2) === 2, rp: (f & 4) === 4 }
    out.scope = MCAST_SCOPE[Number((a >> 112n) & 0xFn)] ?? 'none'
    return out
  }
  if (a === 0n || a === 1n)
    out.scope = a === 0n ? 'none' : 'host'
  else if (a >> 118n === 0x3FAn)
    out.scope = 'link'
  out.iid = iidOf(a)
  out.prefix64 = `${formatIPv6(a & ~LOW64 & ((1n << 128n) - 1n))}/64`
  out.solicitedNode = formatIPv6(0xFF0200000000000000000001FF000000n | (a & 0xFFFFFFn))
  return out
}
