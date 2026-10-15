import type { components } from '@/api/schema'

export type CommonPeering = components['schemas']['CommonPeering']
export type PeeringPort = components['schemas']['PeeringPort']

/** "100G, 10G (RS)" */
export function ports(list: PeeringPort[]): string {
  return list.map(p => `${p.speedMbps >= 1000 ? `${p.speedMbps / 1000}G` : `${p.speedMbps}M`}${p.rsPeer ? ' (RS)' : ''}`).join(', ')
}
