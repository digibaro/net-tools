import type { components } from '@/api/schema'

export type PmtuResult = components['schemas']['PmtuResult']
export interface PmtuProbe { mtu: number, ok: boolean }

/** Common reasons for a given MTU, to explain the result. */
export function likelyCause(mtu: number): string | undefined {
  const known: Record<number, string> = { 1500: 'ethernet', 1492: 'pppoe', 1480: 'ipip', 1476: 'gre', 1460: 'ipv6-tunnel', 1420: 'wireguard', 1400: 'vpn', 1280: 'ipv6-minimum', 9000: 'jumbo' }
  return known[mtu]
}
