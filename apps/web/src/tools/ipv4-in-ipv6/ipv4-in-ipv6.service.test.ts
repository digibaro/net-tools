import { describe, expect, it } from 'vitest'
import { parseIPv4, parseIPv6 } from '@/utils/ip'
import { embed, extract, extractCustom, mixed, nat64Embed, nat64Extract } from './ipv4-in-ipv6.service'

const v6 = (s: string) => parseIPv6(s)!
const v4 = (s: string) => parseIPv4(s)!

describe('rFC 6052 examples (§2.4, IPv4 192.0.2.33)', () => {
  it.each([
    ['2001:db8::', 32, '2001:db8:c000:221::'],
    ['2001:db8:100::', 40, '2001:db8:1c0:2:21::'],
    ['2001:db8:122::', 48, '2001:db8:122:c000:2:2100::'],
    ['2001:db8:122:300::', 56, '2001:db8:122:3c0:0:221::'],
    ['2001:db8:122:344::', 64, '2001:db8:122:344:c0:2:2100:0'],
    ['2001:db8:122:344::', 96, '2001:db8:122:344::c000:221'],
  ] as const)('%s/%d', (prefix, len, want) => {
    const a = nat64Embed(v6(prefix), len, v4('192.0.2.33'))
    expect(a).toBe(v6(want))
    expect(nat64Extract(a, len)).toBe(v4('192.0.2.33'))
  })
})

describe('mixed', () => {
  it('formats the last 32 bits as IPv4', () => {
    expect(mixed(v6('::ffff:c000:201'))).toBe('::ffff:192.0.2.1')
    expect(mixed(v6('64:ff9b::c000:201'))).toBe('64:ff9b::192.0.2.1')
    expect(mixed(v6('::c000:201'))).toBe('::192.0.2.1')
    expect(mixed(v6('2001:db8:1:2:3:4:c000:201'))).toBe('2001:db8:1:2:3:4:192.0.2.1')
  })
})

describe('embed', () => {
  it('lists the standard forms', () => {
    const r = embed('192.0.2.1')!
    expect(Object.fromEntries(r.forms.map(f => [f.id, f.address]))).toEqual({
      mapped: '::ffff:192.0.2.1',
      nat64: '64:ff9b::192.0.2.1',
      sixToFour: '2002:c000:201::/48',
      compat: '::192.0.2.1',
    })
    expect(r.prefixError).toBe(false)
  })
  it('adds a custom NAT64 prefix', () => {
    const r = embed('192.0.2.33', '2001:db8:122::/48')!
    expect(r.forms.find(f => f.id === 'nat64Custom')?.address).toBe('2001:db8:122:c000:2:2100::')
    expect(embed('192.0.2.33', '2001:db8::/50')!.prefixError).toBe(true)
    expect(embed('nope')).toBeUndefined()
  })
})

describe('extract', () => {
  it('detects well-known embeddings', () => {
    expect(extract('::ffff:192.0.2.1')).toEqual([{ kind: 'mapped', ipv4: '192.0.2.1' }])
    expect(extract('64:ff9b::c633:6401')).toEqual([{ kind: 'nat64', ipv4: '198.51.100.1' }])
    expect(extract('2002:c000:201::1')).toEqual([{ kind: 'sixToFour', ipv4: '192.0.2.1' }])
    // RFC 4380 §4 example: server 65.54.227.120, client 192.0.2.45 port 40000
    expect(extract('2001:0:4136:e378:8000:63bf:3fff:fdd2')).toEqual([{ kind: 'teredo', ipv4: '192.0.2.45', detail: 'server 65.54.227.120, port 40000' }])
    expect(extract('2001:db8::1')).toEqual([])
    expect(extract('::1')).toEqual([])
    expect(extract('1.2.3.4')).toBeUndefined()
  })
  it('extracts with a custom prefix', () => {
    expect(extractCustom('2001:db8:122:c000:2:2100::', '2001:db8:122::/48')).toBe('192.0.2.33')
    expect(extractCustom('2001:db9::1', '2001:db8:122::/48')).toBeUndefined()
  })
})
