import { describe, expect, it } from 'vitest'
import { filterIcmp, filterProtocols, ICMP, PROTOCOLS } from './protocol-reference.service'

describe('protocol reference', () => {
  it('finds IP protocols by number and name', () => {
    expect(filterProtocols('6')[0]).toMatchObject({ keyword: 'TCP', rfc: ['RFC 9293'] })
    expect(filterProtocols('17')[0]?.keyword).toBe('UDP')
    expect(filterProtocols('esp').map(p => p.number)).toContain(50)
    expect(filterProtocols('').length).toBe(PROTOCOLS.length)
  })

  it('has ICMP types with codes', () => {
    const unreachable = filterIcmp('3', ICMP.v4)[0]!
    expect(unreachable.name).toBe('Destination Unreachable')
    expect(unreachable.codes.find(c => c.code === 4)?.name).toMatch(/Fragmentation Needed/)
  })

  it('searches ICMPv6 including code names', () => {
    expect(filterIcmp('128', ICMP.v6)[0]?.name).toBe('Echo Request')
    expect(filterIcmp('hop limit', ICMP.v6).map(t => t.type)).toEqual([3])
  })
})
