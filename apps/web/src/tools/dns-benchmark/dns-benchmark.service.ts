import type { components } from '@/api/schema'

export type BenchResolver = components['schemas']['DnsBenchResolver']
export type BenchSummary = components['schemas']['DnsBenchSummary']

/** Live order while results stream in: answered first, then by median. */
export function liveOrder(list: BenchResolver[]): BenchResolver[] {
  return [...list].sort((a, b) => (a.answered === 0 ? 1 : 0) - (b.answered === 0 ? 1 : 0) || a.medianMs - b.medianMs)
}

export const successPct = (r: BenchResolver) => (r.queries ? Math.round(r.answered / r.queries * 100) : 0)
