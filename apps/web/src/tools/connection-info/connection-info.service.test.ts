import { describe, expect, it } from 'vitest'
import { connectionEstimate, pageTiming, protocolName, stack } from './connection-info.service'

describe('connection-info', () => {
  it('reads the Network Information API when present', () => {
    expect(connectionEstimate({ connection: { effectiveType: '4g', downlink: 10, rtt: 50, saveData: false } } as unknown as Navigator)).toEqual({ type: undefined, effectiveType: '4g', downlinkMbps: 10, rttMs: 50, saveData: false })
    expect(connectionEstimate({} as Navigator)).toBeUndefined()
  })
  it('computes page timings', () => {
    const e = { nextHopProtocol: 'h2', domainLookupStart: 1, domainLookupEnd: 11, connectStart: 11, secureConnectionStart: 20, connectEnd: 41, requestStart: 41, responseStart: 91.04, responseEnd: 100 } as PerformanceNavigationTiming
    expect(pageTiming(e)).toEqual({ protocol: 'HTTP/2', dnsMs: 10, connectMs: 30, tlsMs: 21, ttfbMs: 50, downloadMs: 9 })
    expect(pageTiming(undefined)).toBeUndefined()
  })
  it('names protocols and stacks', () => {
    expect(protocolName('h3')).toBe('HTTP/3 (QUIC)')
    expect(stack('1.2.3.4', '2001:db8::1')).toBe('dual')
    expect(stack(undefined, '2001:db8::1')).toBe('v6-only')
    expect(stack()).toBe('none')
  })
})
