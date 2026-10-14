import ipaddr from 'ipaddr.js'

/** Shared IPv4/IPv6 math. Addresses are BigInt for both families so code paths are uniform. */
export type IPVersion = 4 | 6

export interface IPAddress {
  version: IPVersion
  value: bigint
}

export interface Cidr {
  version: IPVersion
  /** Address as entered (may have host bits set). */
  address: bigint
  /** Network address (host bits cleared). */
  network: bigint
  prefix: number
}

const OCTET = '(?:25[0-5]|2[0-4]\\d|1\\d\\d|[1-9]?\\d)'
const V4_RE = new RegExp(`^${OCTET}(?:\\.${OCTET}){3}$`)

export const bitLength = (v: IPVersion) => (v === 4 ? 32 : 128)
export const maxValue = (v: IPVersion) => (1n << BigInt(bitLength(v))) - 1n

/** Strict dotted-quad (no leading zeros, no shorthand). */
export function parseIPv4(input: string): bigint | undefined {
  const s = input.trim()
  if (!V4_RE.test(s))
    return undefined
  return s.split('.').reduce((acc, o) => (acc << 8n) | BigInt(Number(o)), 0n)
}

export function formatIPv4(value: bigint): string {
  return [24n, 16n, 8n, 0n].map(s => String((value >> s) & 0xFFn)).join('.')
}

/** IPv6 incl. embedded IPv4 (::ffff:1.2.3.4); zone ids are rejected. */
export function parseIPv6(input: string): bigint | undefined {
  const s = input.trim().replace(/^\[(.*)\]$/, '$1')
  if (!s.includes(':') || s.includes('%') || !ipaddr.IPv6.isValid(s))
    return undefined
  return ipaddr.IPv6.parse(s).toByteArray().reduce((acc, b) => (acc << 8n) | BigInt(b), 0n)
}

function v6Parts(value: bigint): number[] {
  return Array.from({ length: 8 }, (_, i) => Number((value >> BigInt((7 - i) * 16)) & 0xFFFFn))
}

/** RFC 5952 canonical text form. */
export function formatIPv6(value: bigint): string {
  return new ipaddr.IPv6(v6Parts(value)).toRFC5952String()
}

/** Full form, eight 4-digit groups. */
export function expandIPv6(value: bigint): string {
  return v6Parts(value).map(p => p.toString(16).padStart(4, '0')).join(':')
}

export function parseIP(input: string): IPAddress | undefined {
  const v4 = parseIPv4(input)
  if (v4 !== undefined)
    return { version: 4, value: v4 }
  const v6 = parseIPv6(input)
  return v6 === undefined ? undefined : { version: 6, value: v6 }
}

export function formatIP(version: IPVersion, value: bigint): string {
  return version === 4 ? formatIPv4(value) : formatIPv6(value)
}

export function prefixMask(version: IPVersion, prefix: number): bigint {
  const bits = bitLength(version)
  return prefix === 0 ? 0n : (maxValue(version) >> BigInt(bits - prefix)) << BigInt(bits - prefix)
}

export function hostMask(version: IPVersion, prefix: number): bigint {
  return maxValue(version) ^ prefixMask(version, prefix)
}

/** Parses "addr/len"; a bare address becomes a host route (/32 or /128) when allowBare. */
export function parseCidr(input: string, { allowBare = true } = {}): Cidr | undefined {
  const s = input.trim()
  const slash = s.indexOf('/')
  const addrPart = slash === -1 ? s : s.slice(0, slash)
  const addr = parseIP(addrPart)
  if (!addr)
    return undefined
  let prefix = bitLength(addr.version)
  if (slash !== -1) {
    const p = s.slice(slash + 1)
    if (!/^\d{1,3}$/.test(p) || Number(p) > bitLength(addr.version))
      return undefined
    prefix = Number(p)
  }
  else if (!allowBare) {
    return undefined
  }
  return { version: addr.version, address: addr.value, network: addr.value & prefixMask(addr.version, prefix), prefix }
}

export function cidrLast(c: Pick<Cidr, 'version' | 'network' | 'prefix'>): bigint {
  return c.network | hostMask(c.version, c.prefix)
}

export function cidrSize(c: Pick<Cidr, 'version' | 'prefix'>): bigint {
  return 1n << BigInt(bitLength(c.version) - c.prefix)
}

export function cidrToString(c: Pick<Cidr, 'version' | 'network' | 'prefix'>): string {
  return `${formatIP(c.version, c.network)}/${c.prefix}`
}

export function cidrContains(c: Pick<Cidr, 'version' | 'network' | 'prefix'>, a: IPAddress): boolean {
  return a.version === c.version && (a.value & prefixMask(c.version, c.prefix)) === c.network
}

function makeCidr(version: IPVersion, network: bigint, prefix: number): Cidr {
  return { version, address: network, network, prefix }
}

/** Minimal list of CIDR blocks exactly covering [start, end]. */
export function rangeToCidrs(version: IPVersion, start: bigint, end: bigint): Cidr[] {
  if (start > end)
    return []
  const bits = bitLength(version)
  const out: Cidr[] = []
  let cur = start
  while (cur <= end) {
    // Largest block aligned at cur...
    let size = cur === 0n ? 1n << BigInt(bits) : cur & -cur
    // ...that does not overshoot end.
    while (size > end - cur + 1n)
      size >>= 1n
    const prefix = bits - (size.toString(2).length - 1)
    out.push(makeCidr(version, cur, prefix))
    cur += size
  }
  return out
}

/** Merges overlapping and adjacent prefixes into the minimal covering set (per family). */
export function aggregateCidrs(cidrs: Cidr[]): Cidr[] {
  const out: Cidr[] = []
  for (const version of [4, 6] as const) {
    const ranges = cidrs
      .filter(c => c.version === version)
      .map(c => [c.network, cidrLast(c)] as [bigint, bigint])
      .sort((a, b) => (a[0] < b[0] ? -1 : a[0] > b[0] ? 1 : 0))
    const merged: [bigint, bigint][] = []
    for (const r of ranges) {
      const last = merged.at(-1)
      if (last && r[0] <= last[1] + 1n)
        last[1] = r[1] > last[1] ? r[1] : last[1]
      else
        merged.push([r[0], r[1]])
    }
    for (const [s, e] of merged)
      out.push(...rangeToCidrs(version, s, e))
  }
  return out
}

/** Thousands-separated BigInt, plus a power-of-two hint for huge numbers. */
export function formatCount(n: bigint): string {
  const s = n.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  if (n >= 1n << 40n && (n & (n - 1n)) === 0n)
    return `2^${n.toString(2).length - 1} (${s})`
  return s
}

/** Splits any whitespace/comma separated list into trimmed entries. */
export function splitList(input: string): string[] {
  return input.split(/[\s,;]+/).map(s => s.trim()).filter(Boolean)
}
