import type { components } from '@/api/schema'
import { asnPath } from '@/utils/bgp'

export type AsnInfo = components['schemas']['AsnInfo']
export { parseAsn } from '@/utils/bgp'

export function neighbourRows(info: AsnInfo, label: (type: string) => string) {
  return info.neighbours.map(n => ({ asn: `AS${n.asn}`, link: asnPath(n.asn), relation: label(n.type), power: n.power }))
}
