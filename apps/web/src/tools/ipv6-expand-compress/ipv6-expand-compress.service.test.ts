import { describe, expect, it } from 'vitest'
import { ip6Arpa, ipv6Forms } from './ipv6-expand-compress.service'

describe('ipv6Forms', () => {
  it('produces canonical and expanded forms', () => {
    expect(ipv6Forms('2001:0DB8:0000:0000:0001:0000:0000:0001')).toEqual({
      compressed: '2001:db8::1:0:0:1',
      expanded: '2001:0db8:0000:0000:0001:0000:0000:0001',
      expandedNoZeros: '2001:db8:0:0:1:0:0:1',
      reverse: '1.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa',
      uri: '[2001:db8::1:0:0:1]',
      hex: '0x20010db8000000000001000000000001',
      decimal: '42540766411282592856904266426630537217',
      type: 'Documentation (RFC 3849)',
    })
  })

  it('applies RFC 5952 rules (longest zero run, single 0 not compressed)', () => {
    expect(ipv6Forms('2001:db8:0:1:1:1:1:1')?.compressed).toBe('2001:db8:0:1:1:1:1:1')
    expect(ipv6Forms('2001:0:0:1:0:0:0:1')?.compressed).toBe('2001:0:0:1::1')
  })

  it.each(['1.2.3.4', 'fe80::1%eth0', ':::', ''])('rejects %j', (s) => {
    expect(ipv6Forms(s)).toBeUndefined()
  })

  it('ip6Arpa for ::1', () => {
    expect(ip6Arpa(1n)).toBe(`1${'.0'.repeat(31)}.ip6.arpa`)
  })
})
