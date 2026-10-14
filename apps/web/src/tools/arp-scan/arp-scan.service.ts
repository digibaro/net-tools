import type { components } from '@/api/schema'
import { parseCidr } from '@/utils/ip'

export type ArpScanSummary = components['schemas']['ArpScanSummary']
export interface ArpHost { ip: string, mac: string }

/** IPv4 subnet of /22 or smaller (the server also checks it is directly connected). */
export function isScannable(s: string): boolean {
  const c = parseCidr(s.trim(), { allowBare: false })
  return !!c && c.version === 4 && c.prefix >= 22
}
