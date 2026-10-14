import type { RemoteProbeResult } from '@/composables/useRemote'
import { describe, expect, it } from 'vitest'
import { hostOf, overall } from './is-it-down.service'

const probe = (rcv: number): RemoteProbeResult => ({ index: 0, status: 'finished', probe: { continent: 'EU', country: 'DE', city: 'X', asn: 1, network: 'n', latitude: 0, longitude: 0, tags: [] }, result: { status: 'finished', stats: { loss: rcv ? 0 : 100, rcv } } })

describe('is-it-down', () => {
  it('summarizes probes', () => {
    expect(overall('ping', [probe(3), probe(3)])).toBe('up')
    expect(overall('ping', [probe(3), probe(0)])).toBe('partial')
    expect(overall('ping', [probe(0)])).toBe('down')
    expect(overall('ping', [])).toBeUndefined()
  })
  it('extracts the host', () => {
    expect(hostOf('https://www.example.com/path?q=1')).toBe('www.example.com')
    expect(hostOf('example.com/foo')).toBe('example.com')
    expect(hostOf(' 1.1.1.1 ')).toBe('1.1.1.1')
  })
})
