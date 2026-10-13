/** Network Information API (Chromium only); fields are coarse, privacy-rounded estimates. */
interface NetworkInformation {
  type?: string
  effectiveType?: string
  downlink?: number
  rtt?: number
  saveData?: boolean
}

export interface ConnectionEstimate {
  type?: string
  effectiveType?: string
  downlinkMbps?: number
  rttMs?: number
  saveData?: boolean
}

export function connectionEstimate(nav: Navigator & { connection?: NetworkInformation } = navigator): ConnectionEstimate | undefined {
  const c = nav.connection
  if (!c)
    return undefined
  return { type: c.type, effectiveType: c.effectiveType, downlinkMbps: c.downlink, rttMs: c.rtt, saveData: c.saveData }
}

export interface PageTiming {
  protocol: string
  dnsMs: number
  connectMs: number
  tlsMs: number
  ttfbMs: number
  downloadMs: number
}

const PROTOCOLS: Record<string, string> = { 'h3': 'HTTP/3 (QUIC)', 'h2': 'HTTP/2', 'http/1.1': 'HTTP/1.1', 'http/1.0': 'HTTP/1.0' }

export const protocolName = (p: string) => PROTOCOLS[p] ?? (p || '')

const ms = (n: number) => Math.max(0, Math.round(n * 10) / 10)

/** Timings of the page load itself, from the Navigation Timing API. */
export function pageTiming(e: PerformanceNavigationTiming | undefined): PageTiming | undefined {
  if (!e)
    return undefined
  return {
    protocol: protocolName(e.nextHopProtocol),
    dnsMs: ms(e.domainLookupEnd - e.domainLookupStart),
    connectMs: ms(e.connectEnd - e.connectStart),
    tlsMs: e.secureConnectionStart > 0 ? ms(e.connectEnd - e.secureConnectionStart) : 0,
    ttfbMs: ms(e.responseStart - e.requestStart),
    downloadMs: ms(e.responseEnd - e.responseStart),
  }
}

export type Stack = 'dual' | 'v4-only' | 'v6-only' | 'none'

export function stack(v4?: string, v6?: string): Stack {
  if (v4 && v6)
    return 'dual'
  return v4 ? 'v4-only' : v6 ? 'v6-only' : 'none'
}
