import type { components } from '@/api/schema'

export type Ipv6Readiness = components['schemas']['Ipv6Readiness']
export type Ipv6Host = Ipv6Readiness['hosts'][number]

export function hostState(h: Ipv6Host): 'ok' | 'no-aaaa' | 'unreachable' | 'untested' {
  if (!h.aaaa.length)
    return 'no-aaaa'
  if (h.reachable === null)
    return 'untested'
  return h.reachable ? 'ok' : 'unreachable'
}
