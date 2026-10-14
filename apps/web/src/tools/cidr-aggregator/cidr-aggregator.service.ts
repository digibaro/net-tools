import type { Cidr } from '@/utils/ip'
import { aggregateCidrs, cidrToString, parseCidr, splitList } from '@/utils/ip'

export interface AggregationResult {
  output: string[]
  invalid: string[]
  inputCount: number
}

/** Max entries accepted, to keep the page responsive. */
export const MAX_ENTRIES = 50_000

export function aggregate(input: string): AggregationResult {
  const entries = splitList(input).slice(0, MAX_ENTRIES)
  const valid: Cidr[] = []
  const invalid: string[] = []
  for (const e of entries) {
    const c = parseCidr(e)
    if (c)
      valid.push(c)
    else
      invalid.push(e)
  }
  return { output: aggregateCidrs(valid).map(cidrToString), invalid, inputCount: valid.length }
}
