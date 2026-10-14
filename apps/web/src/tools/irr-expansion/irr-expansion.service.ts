import type { PrefixEntry } from '../router-snippets/router-snippets.service'
import type { components } from '@/api/schema'
import { parseCidr } from '@/utils/ip'

export type IrrResult = components['schemas']['IrrResult']

export const SOURCES = ['RPKI', 'RIPE', 'ARIN', 'APNIC', 'AFRINIC', 'LACNIC', 'RADB', 'NTTCOM', 'LEVEL3', 'ALTDB', 'IDNIC', 'JPIRR'] as const

/** AS number or (hierarchical) as-set / route-set name, as accepted by the API. */
export const isSetName = (s: string) => s.length <= 128 && /^(?:AS\d{1,10}|(?:(?:AS\d{1,10}|AS-[\w-]+|RS-[\w-]+):)*(?:AS|RS)-[\w-]+)$/i.test(s.trim())

/** bgpq4 prefixes → snippet entries (ranges kept). */
export function toEntries(r: IrrResult): PrefixEntry[] {
  const out: PrefixEntry[] = []
  for (const p of r.prefixes) {
    const cidr = parseCidr(p.prefix, { allowBare: false })
    if (cidr)
      out.push(p.exact ? { cidr } : { cidr, ge: p.greaterEqual, le: p.lessEqual })
  }
  return out
}

/** "AS65000:AS-CUSTOMERS" → "AS65000-AS-CUSTOMERS" (a valid list name on every platform). */
export const listName = (set: string) => set.toUpperCase().replace(/[^\w-]/g, '-')

/** Plain text: one prefix per line with its bounds. */
export function plain(r: IrrResult): string {
  if (r.mode === 'asns')
    return r.asns.map(a => `AS${a}`).join('\n')
  return r.prefixes.map(p => p.exact ? p.prefix : `${p.prefix} ${p.greaterEqual}-${p.lessEqual}`).join('\n')
}
