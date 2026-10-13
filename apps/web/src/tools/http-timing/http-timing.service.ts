import type { components } from '@/api/schema'

export type HttpHops = components['schemas']['HttpHops']
type Timing = components['schemas']['HttpHop']['timing']

export interface Phase { id: 'resolve' | 'connect' | 'tls' | 'ttfb' | 'transfer', ms: number, startMs: number }

/** Sequential phases (curl -w style); "ttfb" is server wait after the request was sent. */
export function phases(t: Timing): Phase[] {
  const wait = Math.max(0, t.ttfbMs - t.connectMs - t.tlsMs)
  const list: [Phase['id'], number][] = [['resolve', t.resolveMs], ['connect', t.connectMs], ['tls', t.tlsMs], ['ttfb', wait], ['transfer', t.transferMs]]
  let at = 0
  return list.map(([id, ms]) => {
    const p = { id, ms: Math.round(ms * 1000) / 1000, startMs: Math.round(at * 1000) / 1000 }
    at += ms
    return p
  })
}

/** Rough rating of server response time (time to first byte). */
export function ttfbRating(ttfbMs: number): 'fast' | 'ok' | 'slow' {
  return ttfbMs < 200 ? 'fast' : ttfbMs < 800 ? 'ok' : 'slow'
}
