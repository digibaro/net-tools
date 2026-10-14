import type { Cidr } from '@/utils/ip'
import { bitLength, cidrLast, cidrSize, formatCount, formatIP, parseCidr } from '@/utils/ip'

export const MAX_ROWS = 1024

export interface SplitRow {
  index: number
  cidr: string
  first: string
  last: string
  addresses: string
}

export interface SplitResult {
  newPrefix: number
  total: string
  rows: SplitRow[]
  truncated: boolean
}

export type SplitMode = 'prefix' | 'count'

/** Prefix length needed to split `prefix` into at least `count` equal subnets. */
export function prefixForCount(prefix: number, count: number): number {
  return prefix + Math.ceil(Math.log2(Math.max(1, count)))
}

export function split(input: string, mode: SplitMode, value: number): SplitResult | undefined {
  const c = parseCidr(input, { allowBare: false })
  if (!c || !Number.isInteger(value) || value < 1)
    return undefined
  const newPrefix = mode === 'count' ? prefixForCount(c.prefix, value) : value
  if (newPrefix < c.prefix || newPrefix > bitLength(c.version))
    return undefined
  const totalCount = 1n << BigInt(newPrefix - c.prefix)
  const step = 1n << BigInt(bitLength(c.version) - newPrefix)
  const n = totalCount > BigInt(MAX_ROWS) ? MAX_ROWS : Number(totalCount)
  const rows: SplitRow[] = []
  for (let i = 0; i < n; i++) {
    const sub: Cidr = { version: c.version, address: 0n, network: c.network + BigInt(i) * step, prefix: newPrefix }
    rows.push({
      index: i + 1,
      cidr: `${formatIP(c.version, sub.network)}/${newPrefix}`,
      first: formatIP(c.version, sub.network),
      last: formatIP(c.version, cidrLast(sub)),
      addresses: formatCount(cidrSize(sub)),
    })
  }
  return { newPrefix, total: formatCount(totalCount), rows, truncated: totalCount > BigInt(MAX_ROWS) }
}
