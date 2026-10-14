import type { components } from '@/api/schema'

export type PeeringDbNetwork = components['schemas']['PeeringDbNetwork']
export { parseAsn } from '@/utils/bgp'

/** 100000 → "100G", 10000 → "10G", 1000 → "1G", 500 → "500M". */
export function portSpeed(mbps: number): string {
  if (mbps >= 1000)
    return `${Math.round(mbps / 100) / 10}G`.replace('.0G', 'G')
  return `${mbps}M`
}
