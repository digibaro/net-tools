import type { components } from '@/api/schema'

export type PrefixStatus = components['schemas']['PrefixStatus']
export { isPrefixOrIp } from '@/utils/bgp'

/** Share of RIS peers seeing the prefix, 0–100. */
export const visibilityPct = (s: PrefixStatus) => (s.totalPeers ? Math.round(s.seenBy / s.totalPeers * 1000) / 10 : 0)
