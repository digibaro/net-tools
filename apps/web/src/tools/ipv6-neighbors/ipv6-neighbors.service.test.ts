import type { NdpRouter } from './ipv6-neighbors.service'
import { describe, expect, it } from 'vitest'
import { addressMode, lifetime } from './ipv6-neighbors.service'

const r = (managed: boolean, other: boolean, slaac: boolean) => ({ managed, other, prefixes: [{ prefix: '2001:db8::/64', slaac, onLink: true, validSeconds: 1, preferredSeconds: 1 }] }) as NdpRouter

describe('ipv6-neighbors', () => {
  it('derives the address mode from the RA', () => {
    expect(addressMode(r(false, false, true))).toBe('slaac')
    expect(addressMode(r(false, true, true))).toBe('slaac+dhcpv6-info')
    expect(addressMode(r(true, true, false))).toBe('dhcpv6')
    expect(addressMode(r(false, false, false))).toBe('none')
  })
  it('formats lifetimes', () => {
    expect(lifetime(86400)).toBe('1 d')
    expect(lifetime(1800)).toBe('30 min')
    expect(lifetime(14400)).toBe('4 h')
    expect(lifetime(0xFFFFFFFF)).toBe('∞')
  })
})
