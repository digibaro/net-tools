import { parseIPv4 } from '@/utils/ip'
import { formatMac, parseMac } from '@/utils/mac'

/** Any common MAC notation → aa:bb:cc:dd:ee:ff (shared parser in utils/mac). */
export function normalizeMac(input: string): string | undefined {
  const m = parseMac(input)
  return m === undefined ? undefined : formatMac(m)
}

/** Empty (limited broadcast) or an IPv4 broadcast address in private space. */
export function isBroadcast(s: string): boolean {
  const v = s.trim()
  if (!v || v === '255.255.255.255')
    return true
  const n = parseIPv4(v)
  if (n === undefined)
    return false
  const a = Number(n >> 24n)
  const b = Number((n >> 16n) & 255n)
  return a === 10 || (a === 172 && b >= 16 && b <= 31) || (a === 192 && b === 168) || (a === 169 && b === 254)
}
