import { describe, expect, it } from 'vitest'
import { parseCandidate, verdict } from './webrtc-leak-test.service'

describe('webrtc-leak-test', () => {
  it('parses candidates', () => {
    expect(parseCandidate('candidate:842163049 1 udp 1677729535 9.9.9.7 50123 typ srflx raddr 0.0.0.0 rport 0 generation 0')).toEqual({ type: 'srflx', protocol: 'udp', address: '9.9.9.7', port: 50123, kind: 'public' })
    expect(parseCandidate('candidate:1 1 UDP 2122260223 3f1a2b3c-aaaa-bbbb-cccc-1234567890ab.local 54321 typ host')).toMatchObject({ kind: 'mdns', type: 'host' })
    expect(parseCandidate('candidate:2 1 udp 2122260223 192.168.1.20 54321 typ host')).toMatchObject({ kind: 'private' })
    expect(parseCandidate('garbage')).toBeUndefined()
  })
  it('detects leaks', () => {
    const c = [parseCandidate('candidate:2 1 udp 1 192.168.1.20 1 typ host')!, parseCandidate('candidate:3 1 udp 1 8.8.4.4 1 typ srflx')!]
    expect(verdict(c, ['1.0.0.1'])).toEqual({ localLeak: true, publicAddresses: ['8.8.4.4'], mismatched: ['8.8.4.4'] })
    expect(verdict(c, ['8.8.4.4']).mismatched).toEqual([])
  })
})
