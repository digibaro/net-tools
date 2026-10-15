import type { components } from '@/api/schema'

export type RouteHistory = components['schemas']['RouteHistory']

/** Share of RIS full-table peers that see the prefix now (0–100), or undefined without data. */
export const visibility = (h: RouteHistory) => (h.fullTablePeers ? Math.round(h.seeing / h.fullTablePeers * 1000) / 10 : undefined)

/** More than one origin now = MOAS (anycast, or a possible hijack); several over time = history. */
export function originVerdict(h: RouteHistory): 'none' | 'single' | 'moas' {
  const current = h.origins.filter(o => o.current).length
  return current === 0 ? 'none' : current === 1 ? 'single' : 'moas'
}

export const day = (s: string) => s.slice(0, 10)
