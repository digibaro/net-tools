import { describe, expect, it } from 'vitest'
import { parseCidr, parseIP } from './ip'
import { classify, SPECIAL_RANGES } from './special-ranges'

const cl = (s: string) => classify(parseIP(s)!)

describe('special ranges', () => {
  it('all registry entries parse and are network-aligned', () => {
    for (const r of SPECIAL_RANGES) {
      const c = parseCidr(r.cidr)
      expect(c, r.cidr).toBeDefined()
      expect(c!.network, r.cidr).toBe(c!.address)
    }
  })

  it.each([
    ['10.1.2.3', 'private', true],
    ['172.31.255.255', 'private', true],
    ['172.32.0.1', undefined, false],
    ['100.64.0.1', 'shared', true],
    ['127.0.0.1', 'loopback', true],
    ['169.254.169.254', 'link-local', true],
    ['192.0.2.55', 'documentation', true],
    ['192.0.0.9', 'anycast', false],
    ['192.0.0.5', 'protocol', true],
    ['224.0.0.251', 'multicast', true],
    ['255.255.255.255', 'broadcast', true],
    ['8.8.8.8', undefined, false],
    ['::1', 'loopback', true],
    ['::ffff:10.0.0.1', 'translation', true],
    ['2001:db8::1', 'documentation', true],
    ['2001::1', 'tunnel', false],
    ['fd12:3456::1', 'unique-local', true],
    ['fe80::1', 'link-local', true],
    ['ff02::1', 'multicast', true],
    ['2606:4700::1111', undefined, false],
    ['4000::1', undefined, true],
  ])('%s → %s (bogon %s)', (ip, kind, bogon) => {
    const c = cl(ip)
    expect(c.primary?.kind).toBe(kind)
    expect(c.bogon).toBe(bogon)
  })

  it('orders matches most specific first', () => {
    expect(cl('192.0.0.9').matches.map(m => m.cidr)).toEqual(['192.0.0.9/32', '192.0.0.0/24'])
    expect(cl('0.0.0.0').matches.map(m => m.cidr)).toEqual(['0.0.0.0/32', '0.0.0.0/8'])
  })
})
