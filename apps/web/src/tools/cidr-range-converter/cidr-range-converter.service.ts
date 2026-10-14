import type { Cidr } from '@/utils/ip'
import { cidrLast, cidrSize, cidrToString, formatCount, formatIP, parseCidr, parseIP, rangeToCidrs } from '@/utils/ip'

export interface CidrRow {
  cidr: string
  first: string
  last: string
  addresses: string
}

export type ConversionResult
  = | { mode: 'cidr', rows: CidrRow[] }
    | { mode: 'range', rows: CidrRow[], total: string }

export const MAX_ROWS = 512

function toRow(c: Cidr): CidrRow {
  return { cidr: cidrToString(c), first: formatIP(c.version, c.network), last: formatIP(c.version, cidrLast(c)), addresses: formatCount(cidrSize(c)) }
}

/** "a - b" (or "a–b") → minimal CIDR list; "addr/len" → its range. */
export function convert(input: string): ConversionResult | undefined {
  const s = input.trim()
  const range = s.split(/\s*[-–]\s*/)
  if (range.length === 2) {
    const a = parseIP(range[0]!)
    const b = parseIP(range[1]!)
    if (!a || !b || a.version !== b.version || a.value > b.value)
      return undefined
    const cidrs = rangeToCidrs(a.version, a.value, b.value)
    return { mode: 'range', rows: cidrs.slice(0, MAX_ROWS).map(toRow), total: formatCount(b.value - a.value + 1n) }
  }
  const c = s.includes('/') ? parseCidr(s) : undefined
  return c ? { mode: 'cidr', rows: [toRow(c)] } : undefined
}
