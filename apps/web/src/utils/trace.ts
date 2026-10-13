import type { components } from '@/api/schema'

export type TraceHop = components['schemas']['TraceHop']

/** Merges a hop update into the ordered hop list (by hop number). */
export function upsertHop(hops: TraceHop[], h: TraceHop): TraceHop[] {
  const next = hops.filter(x => x.hop !== h.hop)
  next.push(h)
  return next.sort((a, b) => a.hop - b.hop)
}

/** Fills gaps (hops that never answered) so the table reads 1..n. */
export function withGaps(hops: TraceHop[]): TraceHop[] {
  const max = Math.max(0, ...hops.map(h => h.hop))
  return Array.from({ length: max }, (_, i) => hops.find(h => h.hop === i + 1) ?? { hop: i + 1, ip: '', host: '', sent: 0, recv: 0, loss: 100, lastMs: 0, avgMs: 0, bestMs: 0, worstMs: 0, stddevMs: 0, rttMs: [], asn: null })
}

export function traceTextReport(hops: TraceHop[], mtr: boolean): string {
  return withGaps(hops).map((h) => {
    const who = h.ip ? `${h.host && h.host !== h.ip ? `${h.host} (${h.ip})` : h.ip}${h.asn ? ` [AS${h.asn}]` : ''}` : '*'
    return mtr
      ? `${String(h.hop).padStart(2)}. ${who.padEnd(48)} loss ${h.loss}%  snt ${h.sent}  last ${h.lastMs}  avg ${h.avgMs}  best ${h.bestMs}  wrst ${h.worstMs}`
      : `${String(h.hop).padStart(2)}  ${who}  ${h.rttMs.length ? h.rttMs.map(r => `${r} ms`).join('  ') : '* * *'}`
  }).join('\n')
}
