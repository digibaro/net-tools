import type { TraceHop } from '@/utils/trace'

/** AS path of a trace: hop ASNs in order, consecutive duplicates collapsed, unknown hops skipped. */
export function asPath(hops: TraceHop[]): number[] {
  const out: number[] = []
  for (const h of [...hops].sort((a, b) => a.hop - b.hop)) {
    if (h.asn && out.at(-1) !== h.asn)
      out.push(h.asn)
  }
  return out
}

/** Public hop addresses in hop order (one per hop), for geolocation. */
export const hopIps = (hops: TraceHop[]) => [...hops].sort((a, b) => a.hop - b.hop).map(h => h.ip).filter((ip): ip is string => !!ip).slice(0, 64)
