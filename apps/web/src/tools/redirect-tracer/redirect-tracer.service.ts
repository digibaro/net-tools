import type { components } from '@/api/schema'

export type HttpHops = components['schemas']['HttpHops']
export type HttpHop = components['schemas']['HttpHop']

export interface ChainSummary {
  hops: number
  redirects: number
  finalUrl: string
  finalStatus: number
  httpsUpgrade: boolean
  downgrade: boolean
  stoppedBy: string
}

export function summarize(hops: HttpHop[]): ChainSummary {
  const last = hops.at(-1)!
  const schemes = hops.map(h => h.url.split(':')[0])
  return {
    hops: hops.length,
    redirects: hops.filter(h => h.status >= 300 && h.status < 400).length,
    finalUrl: last.url,
    finalStatus: last.status,
    httpsUpgrade: schemes[0] === 'http' && schemes.includes('https'),
    downgrade: schemes.some((s, i) => i > 0 && s === 'http' && schemes[i - 1] === 'https'),
    stoppedBy: last.errorCode,
  }
}
