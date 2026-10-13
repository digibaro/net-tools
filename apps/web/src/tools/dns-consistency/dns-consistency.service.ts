import type { components } from '@/api/schema'

export type DnsConsistency = components['schemas']['DnsConsistency']

export interface Check {
  id: 'serials' | 'nsSets' | 'authoritative' | 'reachable'
  ok: boolean
}

export function checks(r: DnsConsistency): Check[] {
  const reachable = r.servers.length > 0 && r.servers.every(s => !s.error)
  return [
    { id: 'serials', ok: r.serialsConsistent },
    { id: 'nsSets', ok: r.nsSetsConsistent },
    { id: 'authoritative', ok: r.allAuthoritative },
    { id: 'reachable', ok: reachable },
  ]
}
