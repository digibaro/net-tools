import { describe, expect, it } from 'vitest'
import { fromMac, parsePrefix64, toMac } from './eui64-calculator.service'

describe('fromMac', () => {
  it('builds interface ID, link-local and SLAAC addresses', () => {
    expect(fromMac('00:1a:2b:3c:4d:5e', '2001:db8:1:2::/64')).toEqual({
      mac: '00:1a:2b:3c:4d:5e',
      interfaceId: '021a:2bff:fe3c:4d5e',
      linkLocal: 'fe80::21a:2bff:fe3c:4d5e',
      slaac: '2001:db8:1:2:21a:2bff:fe3c:4d5e',
    })
  })

  it('flips the U/L bit back for locally administered MACs', () => {
    expect(fromMac('02:42:ac:11:00:02')?.linkLocal).toBe('fe80::42:acff:fe11:2')
  })

  it('omits SLAAC without a valid /64', () => {
    expect(fromMac('00:1a:2b:3c:4d:5e', '2001:db8::/48')?.slaac).toBeNull()
    expect(fromMac('nope')).toBeUndefined()
  })
})

describe('toMac', () => {
  it('recovers the MAC', () => {
    expect(toMac('fe80::21a:2bff:fe3c:4d5e')).toBe('00:1a:2b:3c:4d:5e')
    expect(toMac('2001:db8::42:acff:fe11:2')).toBe('02:42:ac:11:00:02')
  })
  it('rejects non-EUI-64 interface IDs', () => {
    expect(toMac('2001:db8::1')).toBeUndefined()
  })
})

describe('parsePrefix64', () => {
  it('accepts a /64 with or without length', () => {
    expect(parsePrefix64('2001:db8:1:2::')).toBe(0x2001_0DB8_0001_0002n << 64n)
    expect(parsePrefix64('2001:db8::/56')).toBeUndefined()
    expect(parsePrefix64('')).toBeUndefined()
  })
})
