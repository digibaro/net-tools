import { formatIPv4, formatIPv6, parseCidr, parseIPv4, parseIPv6 } from '@/utils/ip'

/** RFC 6052 §2.2 prefix lengths. */
export const NAT64_LENGTHS = [32, 40, 48, 56, 64, 96] as const

/** Bit positions (0 = most significant) that hold the IPv4 address after a prefix of `len`; bits 64–71 (the "u" octet) are skipped. */
function nat64Positions(len: number): number[] {
  const out: number[] = []
  for (let pos = len; out.length < 32; pos++) {
    if (pos >= 64 && pos < 72)
      continue
    out.push(pos)
  }
  return out
}

/** Embeds v4 into prefix/len per RFC 6052. */
export function nat64Embed(prefix: bigint, len: number, v4: bigint): bigint {
  let addr = prefix
  nat64Positions(len).forEach((pos, i) => {
    if ((v4 >> BigInt(31 - i)) & 1n)
      addr |= 1n << BigInt(127 - pos)
  })
  return addr
}

/** Extracts the IPv4 address from an RFC 6052 address with a prefix of `len`. */
export function nat64Extract(addr: bigint, len: number): bigint {
  let v4 = 0n
  for (const pos of nat64Positions(len))
    v4 = (v4 << 1n) | ((addr >> BigInt(127 - pos)) & 1n)
  return v4
}

/** Mixed notation for addresses whose IPv4 sits in the last 32 bits ("::ffff:192.0.2.1"). */
export function mixed(addr: bigint): string {
  const groups = Array.from({ length: 6 }, (_, i) => Number((addr >> BigInt(112 - i * 16)) & 0xFFFFn))
  // Longest run of two or more zero groups is compressed (RFC 5952).
  let best = -1
  let bestLen = 1
  for (let i = 0; i < 6;) {
    let j = i
    while (j < 6 && groups[j] === 0) j++
    if (j - i > bestLen) {
      best = i
      bestLen = j - i
    }
    i = j === i ? i + 1 : j
  }
  const hex = (g: number[]) => g.map(n => n.toString(16)).join(':')
  const head = best === -1 ? `${hex(groups)}:` : `${hex(groups.slice(0, best))}::${best + bestLen < 6 ? `${hex(groups.slice(best + bestLen))}:` : ''}`
  return head + formatIPv4(addr & 0xFFFFFFFFn)
}

export interface Form { id: string, address: string, note?: string }

/** IPv6 representations that embed an IPv4 address. */
export function embed(v4Input: string, nat64Prefix = '64:ff9b::/96'): { forms: Form[], prefixError: boolean } | undefined {
  const v4 = parseIPv4(v4Input.trim())
  if (v4 === undefined)
    return undefined
  const forms: Form[] = [
    { id: 'mapped', address: mixed(0xFFFFn << 32n | v4) },
    { id: 'nat64', address: mixed(0x64FF9Bn << 96n | v4) },
    { id: 'sixToFour', address: `${formatIPv6(0x2002n << 112n | v4 << 80n).replace(/::$/, '')}::/48` },
    { id: 'compat', address: mixed(v4), note: 'deprecated' },
  ]
  const p = parseCidr(nat64Prefix, { allowBare: false })
  const ok = !!p && p.version === 6 && (NAT64_LENGTHS as readonly number[]).includes(p.prefix) && ((p.network >> 56n) & 0xFFn) === 0n
  if (ok && (p.prefix !== 96 || p.network !== 0x64FF9Bn << 96n)) {
    const a = nat64Embed(p.network, p.prefix, v4)
    forms.splice(2, 0, { id: 'nat64Custom', address: p.prefix === 96 ? mixed(a) : formatIPv6(a), note: `${formatIPv6(p.network)}/${p.prefix}` })
  }
  return { forms, prefixError: !ok }
}

export interface Extracted { kind: string, ipv4: string, detail?: string }

const in6 = (a: bigint, net: bigint, len: number) => (a >> BigInt(128 - len)) === (net >> BigInt(128 - len))

/** Finds IPv4 addresses embedded in an IPv6 address by well-known prefix. */
export function extract(v6Input: string): Extracted[] | undefined {
  const a = parseIPv6(v6Input.trim())
  if (a === undefined)
    return undefined
  const low = formatIPv4(a & 0xFFFFFFFFn)
  const out: Extracted[] = []
  if (in6(a, 0xFFFFn << 32n, 96))
    out.push({ kind: 'mapped', ipv4: low })
  if (in6(a, 0x64FF9Bn << 96n, 96))
    out.push({ kind: 'nat64', ipv4: low })
  if (in6(a, 0x64FF9B0001n << 80n, 48))
    out.push({ kind: 'nat64Local', ipv4: formatIPv4(nat64Extract(a, 48)) })
  if (in6(a, 0x2002n << 112n, 16))
    out.push({ kind: 'sixToFour', ipv4: formatIPv4((a >> 80n) & 0xFFFFFFFFn) })
  if (in6(a, 0x20010000n << 96n, 32)) {
    const server = formatIPv4((a >> 64n) & 0xFFFFFFFFn)
    const port = Number(((a >> 32n) & 0xFFFFn) ^ 0xFFFFn)
    out.push({ kind: 'teredo', ipv4: formatIPv4((a & 0xFFFFFFFFn) ^ 0xFFFFFFFFn), detail: `server ${server}, port ${port}` })
  }
  if (a > 1n && a < 1n << 32n)
    out.push({ kind: 'compat', ipv4: low })
  return out
}

/** Extracts with a user-supplied NAT64 prefix (any RFC 6052 length). */
export function extractCustom(v6Input: string, nat64Prefix: string): string | undefined {
  const a = parseIPv6(v6Input.trim())
  const p = parseCidr(nat64Prefix, { allowBare: false })
  if (a === undefined || !p || p.version !== 6 || !(NAT64_LENGTHS as readonly number[]).includes(p.prefix))
    return undefined
  if (!in6(a, p.network, p.prefix))
    return undefined
  return formatIPv4(nat64Extract(a, p.prefix))
}
