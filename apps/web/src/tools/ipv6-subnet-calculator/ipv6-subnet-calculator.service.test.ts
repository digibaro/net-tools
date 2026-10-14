import { describe, expect, it } from 'vitest'
import { calculateIpv6Subnet, reverseZone } from './ipv6-subnet-calculator.service'

describe('calculateIpv6Subnet', () => {
  it('computes a /48', () => {
    expect(calculateIpv6Subnet('2001:DB8:abcd:12::1/48')).toMatchObject({
      address: '2001:db8:abcd:12::1',
      network: '2001:db8:abcd::/48',
      networkExpanded: '2001:0db8:abcd:0000:0000:0000:0000:0000',
      lastAddress: '2001:db8:abcd:ffff:ffff:ffff:ffff:ffff',
      subnets64: '65,536',
      subnets48: '1',
      totalAddresses: '2^80 (1,208,925,819,614,629,174,706,176)',
      reverseZone: 'd.c.b.a.8.b.d.0.1.0.0.2.ip6.arpa',
      type: 'Documentation (RFC 3849)',
      previousNetwork: '2001:db8:abcc::/48',
      nextNetwork: '2001:db8:abce::/48',
    })
  })

  it('handles /128, /0 and long prefixes', () => {
    expect(calculateIpv6Subnet('::1')).toMatchObject({ prefix: 128, totalAddresses: '1', subnets64: '0', type: 'Loopback Address (RFC 4291)' })
    expect(calculateIpv6Subnet('::/0')).toMatchObject({ previousNetwork: null, nextNetwork: null, reverseZone: 'ip6.arpa' })
    expect(calculateIpv6Subnet('2606:4700::1/127')?.type).toBe('Global unicast')
  })

  it.each(['10.0.0.0/8', '2001:db8::/129', 'nope', ''])('rejects %j', (s) => {
    expect(calculateIpv6Subnet(s)).toBeUndefined()
  })
})

describe('reverseZone', () => {
  it('truncates to whole nibbles', () => {
    expect(reverseZone(0x20010DB8n << 96n, 30)).toBe('b.d.0.1.0.0.2.ip6.arpa')
  })
})
