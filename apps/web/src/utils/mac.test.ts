import { describe, expect, it } from 'vitest'
import { formatMac, isLocallyAdministered, isMulticast, macToEui64, ouiOf, parseMac } from './mac'

describe('parseMac', () => {
  it.each(['00:1A:2B:3C:4D:5E', '00-1a-2b-3c-4d-5e', '001a.2b3c.4d5e', '001a2b3c4d5e', ' 00:1a:2b:3c:4d:5e '])('accepts %j', (s) => {
    expect(parseMac(s)).toBe(0x001A2B3C4D5En)
  })
  it.each(['00:1a:2b:3c:4d', '00:1a-2b:3c:4d:5e', 'zz:1a:2b:3c:4d:5e', '001a.2b3c.4d5', '', '00:1a:2b:3c:4d:5e:6f'])('rejects %j', (s) => {
    expect(parseMac(s)).toBeUndefined()
  })
})

describe('mac helpers', () => {
  it('formats and inspects bits', () => {
    const m = parseMac('02:00:5e:10:00:01')!
    expect(formatMac(m)).toBe('02:00:5e:10:00:01')
    expect(formatMac(m, '-')).toBe('02-00-5e-10-00-01')
    expect(isLocallyAdministered(m)).toBe(true)
    expect(isMulticast(m)).toBe(false)
    expect(isMulticast(parseMac('01:00:5e:00:00:fb')!)).toBe(true)
    expect(ouiOf(parseMac('00:1a:2b:3c:4d:5e')!)).toBe('001A2B')
  })

  it('macToEui64 follows RFC 4291 Appendix A', () => {
    expect(macToEui64(parseMac('00:1a:2b:3c:4d:5e')!).toString(16)).toBe('21a2bfffe3c4d5e')
    expect(macToEui64(parseMac('02:00:00:00:00:01')!).toString(16).padStart(16, '0')).toBe('000000fffe000001')
  })
})
