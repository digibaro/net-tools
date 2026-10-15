import type { components } from '@/api/schema'

export type CdnResult = components['schemas']['CdnResult']
export type Detection = CdnResult['detections'][number]

export const KIND_ORDER = ['cdn', 'waf', 'bot', 'hosting', 'cache', 'tls'] as const

export function sortDetections(d: Detection[]): Detection[] {
  return [...d].sort((a, b) => KIND_ORDER.indexOf(a.kind) - KIND_ORDER.indexOf(b.kind) || b.evidence.length - a.evidence.length)
}
