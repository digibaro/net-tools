import type { Cidr, IPVersion } from '@/utils/ip'
import { bitLength, cidrToString, formatIP, hostMask, parseCidr, splitList } from '@/utils/ip'

/** One prefix-list entry; ge/le are optional length bounds (as in IOS / bgpq4). */
export interface PrefixEntry {
  cidr: Cidr
  ge?: number
  le?: number
}

export type Platform = 'ios' | 'iosxr' | 'junos' | 'eos' | 'bird' | 'routeros'
export type Kind = 'prefix-list' | 'acl'
export type Action = 'permit' | 'deny'

export const PLATFORMS: Platform[] = ['ios', 'iosxr', 'eos', 'junos', 'bird', 'routeros']

export interface ParseResult {
  entries: PrefixEntry[]
  invalid: string[]
}

/** Accepts CIDRs (one per line or comma/space separated); bare addresses become host routes. */
export function parsePrefixes(input: string): ParseResult {
  const entries: PrefixEntry[] = []
  const invalid: string[] = []
  const seen = new Set<string>()
  for (const item of splitList(input)) {
    const c = parseCidr(item)
    if (!c) {
      invalid.push(item)
      continue
    }
    const key = `${c.version}|${c.network}|${c.prefix}`
    if (!seen.has(key)) {
      seen.add(key)
      entries.push({ cidr: c })
    }
  }
  return { entries, invalid }
}

/** Applies an "or longer up to /N" policy (upTo undefined = exact match). */
export function withUpTo(entries: PrefixEntry[], upTo4?: number, upTo6?: number): PrefixEntry[] {
  return entries.map((e) => {
    const upTo = e.cidr.version === 4 ? upTo4 : upTo6
    if (upTo === undefined || upTo <= e.cidr.prefix)
      return { cidr: e.cidr }
    return { cidr: e.cidr, le: Math.min(upTo, bitLength(e.cidr.version)) }
  })
}

/** Normalized bounds: undefined when the entry is an exact match. */
function bounds(e: PrefixEntry): { ge: number, le: number } | undefined {
  const ge = e.ge ?? e.cidr.prefix
  const le = e.le ?? ge
  if (ge === e.cidr.prefix && le === e.cidr.prefix)
    return undefined
  return { ge, le }
}

const s = (e: PrefixEntry) => cidrToString(e.cidr)

/**
 * IOS / EOS ge-le suffix. ge is omitted when it equals the prefix length; le may only be omitted
 * when it is the family maximum, because a lone "ge N" means N up to /32 (/128).
 */
function iosRange(e: PrefixEntry): string {
  const b = bounds(e)
  if (!b)
    return ''
  const ge = b.ge > e.cidr.prefix ? ` ge ${b.ge}` : ''
  const le = ge && b.le === bitLength(e.cidr.version) ? '' : ` le ${b.le}`
  return ge + le
}

function byFamily(entries: PrefixEntry[]): [IPVersion, PrefixEntry[]][] {
  return ([4, 6] as IPVersion[]).map(v => [v, entries.filter(e => e.cidr.version === v)] as [IPVersion, PrefixEntry[]]).filter(([, es]) => es.length > 0)
}

const name6 = (name: string, both: boolean, v: IPVersion) => (both && v === 6 ? `${name}-V6` : name)

/** Prefix-list in the chosen platform's syntax. */
export function prefixList(entries: PrefixEntry[], platform: Platform, name: string, action: Action = 'permit'): string {
  const fams = byFamily(entries)
  const both = fams.length > 1
  const out: string[] = []
  for (const [v, es] of fams) {
    const n = name6(name, both, v)
    switch (platform) {
      case 'ios':
        es.forEach((e, i) => out.push(`${v === 4 ? 'ip' : 'ipv6'} prefix-list ${n} seq ${(i + 1) * 5} ${action} ${s(e)}${iosRange(e)}`))
        break
      case 'eos':
        if (v === 4) {
          es.forEach((e, i) => out.push(`ip prefix-list ${n} seq ${(i + 1) * 10} ${action} ${s(e)}${iosRange(e)}`))
        }
        else { // EOS configures IPv6 prefix-lists in a sub-mode
          out.push(`ipv6 prefix-list ${n}`)
          es.forEach((e, i) => out.push(`   seq ${(i + 1) * 10} ${action} ${s(e)}${iosRange(e)}`))
        }
        break
      case 'iosxr':
        out.push(`prefix-set ${n}`)
        es.forEach((e, i) => {
          const b = bounds(e)
          out.push(`  ${s(e)}${b ? ` ge ${b.ge} le ${b.le}` : ''}${i < es.length - 1 ? ',' : ''}`)
        })
        out.push('end-set')
        break
      case 'junos':
        if (es.every(e => !bounds(e))) {
          es.forEach(e => out.push(`set policy-options prefix-list ${n} ${s(e)}`))
        }
        else {
          es.forEach((e) => {
            const b = bounds(e)
            out.push(`set policy-options route-filter-list ${n} ${s(e)} ${b ? (b.ge === e.cidr.prefix ? `upto /${b.le}` : `prefix-length-range /${b.ge}-/${b.le}`) : 'exact'}`)
          })
        }
        break
      case 'bird':
        out.push(`define ${n.replace(/\W/g, '_').toUpperCase()} = [`)
        es.forEach((e, i) => {
          const b = bounds(e)
          out.push(`  ${s(e)}${b ? `{${b.ge},${b.le}}` : ''}${i < es.length - 1 ? ',' : ''}`)
        })
        out.push('];')
        break
      case 'routeros':
        es.forEach(e => out.push(`/routing filter rule add chain=${n} rule="if (dst in ${s(e)}${bounds(e) ? ` && dst-len in ${bounds(e)!.ge}-${bounds(e)!.le}` : ` && dst-len == ${e.cidr.prefix}`}) { ${action === 'permit' ? 'accept' : 'reject'} }"`))
        break
    }
    if (out.length && fams.length > 1 && v === 4)
      out.push('')
  }
  return out.join('\n')
}

/** Cisco wildcard mask: 10.0.0.0/24 → 0.0.0.255 */
export const wildcard = (c: Cidr) => formatIP(4, hostMask(4, c.prefix))

/** Source-address ACL / filter matching the prefixes. */
export function acl(entries: PrefixEntry[], platform: Platform, name: string, action: Action = 'permit'): string {
  const fams = byFamily(entries)
  const both = fams.length > 1
  const out: string[] = []
  for (const [v, es] of fams) {
    const n = name6(name, both, v)
    switch (platform) {
      case 'ios':
        out.push(v === 4 ? `ip access-list extended ${n}` : `ipv6 access-list ${n}`)
        es.forEach((e) => {
          const src = v === 4 ? (e.cidr.prefix === 32 ? `host ${formatIP(4, e.cidr.network)}` : `${formatIP(4, e.cidr.network)} ${wildcard(e.cidr)}`) : s(e)
          out.push(` ${action} ${v === 4 ? 'ip' : 'ipv6'} ${src} any`)
        })
        break
      case 'eos':
        out.push(`${v === 4 ? 'ip' : 'ipv6'} access-list ${n}`)
        es.forEach((e, i) => out.push(`   ${(i + 1) * 10} ${action} ${v === 4 ? 'ip' : 'ipv6'} ${s(e)} any`))
        break
      case 'iosxr':
        out.push(`${v === 4 ? 'ipv4' : 'ipv6'} access-list ${n}`)
        es.forEach((e, i) => out.push(` ${(i + 1) * 10} ${action} ${v === 4 ? 'ipv4' : 'ipv6'} ${s(e)} any`))
        break
      case 'junos':
        es.forEach(e => out.push(`set firewall family ${v === 4 ? 'inet' : 'inet6'} filter ${n} term match from source-address ${s(e)}`))
        out.push(`set firewall family ${v === 4 ? 'inet' : 'inet6'} filter ${n} term match then ${action === 'permit' ? 'accept' : 'discard'}`)
        break
      case 'bird':
        return prefixList(entries, 'bird', name, action)
      case 'routeros':
        es.forEach(e => out.push(`/${v === 4 ? 'ip' : 'ipv6'} firewall address-list add list=${n} address=${s(e)}`))
        out.push(`/${v === 4 ? 'ip' : 'ipv6'} firewall filter add chain=forward src-address-list=${n} action=${action === 'permit' ? 'accept' : 'drop'}`)
        break
    }
    if (fams.length > 1 && v === 4)
      out.push('')
  }
  return out.join('\n')
}

/** AS-path / ASN-set filter for a list of origin ASNs. */
export function asnFilter(asns: number[], platform: Platform, name: string): string {
  const n = name.replace(/\W/g, '_')
  switch (platform) {
    case 'ios':
    case 'eos':
      return asns.map(a => `ip as-path access-list ${n} permit _${a}$`).join('\n')
    case 'iosxr':
      return [`as-path-set ${n}`, ...asns.map((a, i) => `  originates-from '${a}'${i < asns.length - 1 ? ',' : ''}`), 'end-set'].join('\n')
    case 'junos':
      return `set policy-options as-list ${n} members [ ${asns.join(' ')} ]`
    case 'bird':
      return `define ${n.toUpperCase()} = [ ${asns.join(', ')} ];`
    case 'routeros':
      return `/routing filter num-list add list=${n} range=${asns.join(',')}`
  }
}

/** Valid list name for all platforms. */
export const isListName = (v: string) => /^[A-Z][\w-]{0,62}$/i.test(v)
