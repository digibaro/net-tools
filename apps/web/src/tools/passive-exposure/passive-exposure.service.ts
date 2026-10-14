import type { components } from '@/api/schema'

export type Exposure = components['schemas']['Exposure']
export { isIpAddress } from '../reverse-dns/reverse-dns.service'

export const cveUrl = (id: string) => `https://nvd.nist.gov/vuln/detail/${encodeURIComponent(id)}`
export const portUrl = (port: number) => `/port-reference?q=${port}`
