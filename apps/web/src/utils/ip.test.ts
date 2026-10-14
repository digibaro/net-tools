import { describe, expect, it } from 'vitest'
import { aggregateCidrs, cidrContains, cidrLast, cidrSize, cidrToString, expandIPv6, formatCount, formatIPv4, formatIPv6, parseCidr, parseIP, parseIPv4, parseIPv6, prefixMask, rangeToCidrs, splitList } from './ip'

const c = (s: string) => parseCidr(s)!

describe('parseIPv4 / formatIPv4', () => {
  it.each(['0.0.0.0', '255.255.255.255', '192.168.1.10', ' 10.0.0.1 '])('round-trips %s', (s) => {
    expect(formatIPv4(parseIPv4(s)!)).toBe(s.trim())
  })
  it.each(['256.0.0.1', '1.2.3', '01.2.3.4', '1.2.3.4.5', '', 'a.b.c.d', '1.2.3.4/24'])('rejects %j', (s) => {
    expect(parseIPv4(s)).toBeUndefined()
  })
})

describe('parseIPv6 / formatIPv6', () => {
  it.each([
    ['2001:DB8:0:0:0:0:0:1', '2001:db8::1'],
    ['::', '::'],
    ['[fe80::1]', 'fe80::1'],
    ['2001:db8:0:0:1:0:0:1', '2001:db8::1:0:0:1'],
    ['::ffff:192.0.2.1', '::ffff:c000:201'],
  ])('%s → %s', (input, want) => {
    expect(formatIPv6(parseIPv6(input)!)).toBe(want)
  })
  it.each(['1.2.3.4', 'fe80::1%eth0', '2001:db8:::1', 'gggg::1', ''])('rejects %j', (s) => {
    expect(parseIPv6(s)).toBeUndefined()
  })
  it('expands', () => {
    expect(expandIPv6(parseIPv6('2001:db8::1')!)).toBe('2001:0db8:0000:0000:0000:0000:0000:0001')
  })
})

describe('parseCidr', () => {
  it('masks host bits and keeps the address', () => {
    const x = c('192.168.1.10/24')
    expect(x).toMatchObject({ version: 4, prefix: 24 })
    expect(formatIPv4(x.network)).toBe('192.168.1.0')
    expect(formatIPv4(x.address)).toBe('192.168.1.10')
    expect(formatIPv4(cidrLast(x))).toBe('192.168.1.255')
  })
  it('handles bare addresses and edge prefixes', () => {
    expect(c('10.0.0.1').prefix).toBe(32)
    expect(c('::1').prefix).toBe(128)
    expect(cidrSize(c('0.0.0.0/0'))).toBe(1n << 32n)
    expect(parseCidr('10.0.0.1', { allowBare: false })).toBeUndefined()
  })
  it.each(['10.0.0.0/33', '::/129', '10.0.0.0/', '10.0.0.0/a', 'x/24'])('rejects %j', (s) => {
    expect(parseCidr(s)).toBeUndefined()
  })
  it('prefixMask', () => {
    expect(formatIPv4(prefixMask(4, 0))).toBe('0.0.0.0')
    expect(formatIPv4(prefixMask(4, 20))).toBe('255.255.240.0')
    expect(formatIPv4(prefixMask(4, 32))).toBe('255.255.255.255')
  })
})

describe('rangeToCidrs', () => {
  const r = (a: string, b: string) => {
    const x = parseIP(a)!
    return rangeToCidrs(x.version, x.value, parseIP(b)!.value).map(cidrToString)
  }
  it.each([
    ['10.0.0.0', '10.0.0.255', ['10.0.0.0/24']],
    ['10.0.0.1', '10.0.0.6', ['10.0.0.1/32', '10.0.0.2/31', '10.0.0.4/31', '10.0.0.6/32']],
    ['0.0.0.0', '255.255.255.255', ['0.0.0.0/0']],
    ['192.168.0.0', '192.168.2.255', ['192.168.0.0/23', '192.168.2.0/24']],
    ['2001:db8::', '2001:db8::ffff', ['2001:db8::/112']],
    ['::', 'ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff', ['::/0']],
  ])('%s – %s', (a, b, want) => {
    expect(r(a, b)).toEqual(want)
  })
  it('returns nothing for inverted ranges', () => {
    expect(rangeToCidrs(4, 10n, 5n)).toEqual([])
  })
})

describe('aggregateCidrs', () => {
  it('merges adjacent, overlapping and duplicate prefixes per family', () => {
    const input = ['10.0.0.0/25', '10.0.0.128/25', '10.0.1.0/24', '10.0.0.0/24', '192.168.0.0/24', '2001:db8::/33', '2001:db8:8000::/33', '10.0.0.5/32']
    expect(aggregateCidrs(input.map(c)).map(cidrToString)).toEqual(['10.0.0.0/23', '192.168.0.0/24', '2001:db8::/32'])
  })
  it('keeps non-aligned merges minimal', () => {
    expect(aggregateCidrs(['10.0.1.0/24', '10.0.2.0/24'].map(c)).map(cidrToString)).toEqual(['10.0.1.0/24', '10.0.2.0/24'])
  })
})

describe('helpers', () => {
  it('cidrContains is family aware', () => {
    expect(cidrContains(c('10.0.0.0/8'), parseIP('10.255.0.1')!)).toBe(true)
    expect(cidrContains(c('10.0.0.0/8'), parseIP('11.0.0.1')!)).toBe(false)
    expect(cidrContains(c('::/0'), parseIP('10.0.0.1')!)).toBe(false)
  })
  it('formatCount', () => {
    expect(formatCount(254n)).toBe('254')
    expect(formatCount(16777216n)).toBe('16,777,216')
    expect(formatCount(1n << 64n)).toBe('2^64 (18,446,744,073,709,551,616)')
  })
  it('splitList', () => {
    expect(splitList(' a, b\nc;d  ')).toEqual(['a', 'b', 'c', 'd'])
  })
})
