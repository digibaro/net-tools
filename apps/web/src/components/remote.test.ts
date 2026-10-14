import type { RemoteProbeResult } from '@/composables/useRemote'
import { describe, expect, it } from 'vitest'
import { countryName, hopRows, pingSummary, verdict } from './remote'

type R = RemoteProbeResult['result']

describe('remote helpers', () => {
  it('summarizes ping', () => {
    const r: R = { status: 'finished', stats: { min: 5.568, avg: 5.666, max: 5.79, loss: 0, total: 3, rcv: 3 } }
    expect(pingSummary(r)).toBe('5.6 ms / 5.7 ms / 5.8 ms · 0 % loss')
    expect(verdict('ping', r)).toBe('up')
    expect(verdict('ping', { status: 'finished', stats: { loss: 100, rcv: 0 } })).toBe('down')
    expect(verdict('ping', { status: 'failed' })).toBe('error')
  })
  it('builds hop rows', () => {
    const r: R = { status: 'finished', resolvedAddress: '2.2.2.2', hops: [
      { resolvedAddress: '1.1.1.1', resolvedHostname: '_gateway', timings: [{ rtt: 1 }, { rtt: 3 }] },
      { resolvedAddress: '2.2.2.2', resolvedHostname: '2.2.2.2', asn: [13335], stats: { avg: 5, loss: 0 }, timings: [] },
    ] }
    expect(hopRows(r)).toEqual([
      { hop: 1, host: '_gateway', ip: '1.1.1.1', asn: '', avg: '2 ms', loss: '' },
      { hop: 2, host: '', ip: '2.2.2.2', asn: 'AS13335', avg: '5 ms', loss: '0 %' },
    ])
    expect(verdict('traceroute', r)).toBe('up')
  })
  it('names countries', () => {
    expect(countryName('DE')).toBe('Germany')
  })
})
