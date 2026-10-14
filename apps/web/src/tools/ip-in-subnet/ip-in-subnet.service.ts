import type { Cidr } from '@/utils/ip'
import { cidrContains, cidrToString, formatIP, parseCidr, parseIP, splitList } from '@/utils/ip'

export interface MatchRow {
  ip: string
  matched: boolean
  /** Matching prefixes, most specific first. */
  prefixes: string[]
}

export interface MatchResult {
  rows: MatchRow[]
  invalidIps: string[]
  invalidPrefixes: string[]
}

export const MAX_IPS = 5000

export function matchIps(ipsInput: string, prefixesInput: string): MatchResult {
  const invalidPrefixes: string[] = []
  const prefixes: Cidr[] = []
  for (const p of splitList(prefixesInput)) {
    const c = parseCidr(p)
    if (c)
      prefixes.push(c)
    else
      invalidPrefixes.push(p)
  }
  prefixes.sort((a, b) => b.prefix - a.prefix)

  const invalidIps: string[] = []
  const rows: MatchRow[] = []
  for (const s of splitList(ipsInput).slice(0, MAX_IPS)) {
    const ip = parseIP(s)
    if (!ip) {
      invalidIps.push(s)
      continue
    }
    const hits = prefixes.filter(c => cidrContains(c, ip)).map(cidrToString)
    rows.push({ ip: formatIP(ip.version, ip.value), matched: hits.length > 0, prefixes: [...new Set(hits)] })
  }
  return { rows, invalidIps, invalidPrefixes }
}
