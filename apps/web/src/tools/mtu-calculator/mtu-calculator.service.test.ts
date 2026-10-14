import { describe, expect, it } from 'vitest'
import { calculateMtu, decodeLayers, encodeLayers } from './mtu-calculator.service'

describe('calculateMtu', () => {
  it('plain Ethernet', () => {
    expect(calculateMtu(1500, {})).toEqual({ totalOverhead: 0, innerMtu: 1500, mssIpv4: 1460, mssIpv6: 1440, belowIpv6Minimum: false })
  })

  it('stacks encapsulations', () => {
    expect(calculateMtu(1500, { pppoe: 1 })?.mssIpv4).toBe(1452)
    expect(calculateMtu(1500, { 'wireguard-v4': 1 })?.innerMtu).toBe(1440)
    expect(calculateMtu(1500, { 'gre': 1, 'ipsec-gcm': 1 })?.innerMtu).toBe(1419)
    expect(calculateMtu(9000, { 'vxlan-v4': 1 })?.innerMtu).toBe(8950)
    expect(calculateMtu(1500, { mpls: 3 })?.totalOverhead).toBe(12)
    expect(calculateMtu(1500, { qinq: 5 })?.totalOverhead).toBe(8)
  })

  it('flags IPv6 minimum and rejects nonsense', () => {
    expect(calculateMtu(1300, { 'wireguard-v6': 1 })?.belowIpv6Minimum).toBe(true)
    expect(calculateMtu(40, {})).toBeUndefined()
    expect(calculateMtu(100, { 'ipsec-cbc': 1 })).toBeUndefined()
  })

  it('round-trips layers through the URL form', () => {
    const layers = { pppoe: 1, mpls: 2, vlan: 0 }
    expect(encodeLayers(layers)).toBe('pppoe,mpls:2')
    expect(decodeLayers('pppoe,mpls:2,bogus,vlan:99')).toEqual({ pppoe: 1, mpls: 2, vlan: 16 })
  })
})
