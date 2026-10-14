import type { components } from '@/api/schema'

export type CloudRanges = components['schemas']['CloudRanges']
export { isIpAddress } from '../reverse-dns/reverse-dns.service'

export const failedSources = (r: CloudRanges) => r.sources.filter(s => s.error).map(s => s.name)
