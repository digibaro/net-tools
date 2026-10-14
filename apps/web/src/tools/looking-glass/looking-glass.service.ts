import type { components } from '@/api/schema'

export type LookingGlass = components['schemas']['LookingGlass']
export { isPrefixOrIp } from '@/utils/bgp'

/** Distinct AS paths with how many collector peers see each, most common first. */
export function pathCounts(lg: LookingGlass): { path: string, peers: number }[] {
  const m = new Map<string, number>()
  for (const r of lg.routes)
    m.set(r.asPath, (m.get(r.asPath) ?? 0) + 1)
  return [...m].map(([path, peers]) => ({ path, peers })).sort((a, b) => b.peers - a.peers)
}
