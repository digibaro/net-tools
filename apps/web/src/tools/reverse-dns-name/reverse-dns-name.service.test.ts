import { describe, expect, it } from 'vitest'
import { reverseName, reverseNames } from './reverse-dns-name.service'

describe('reverseName', () => {
  it('builds PTR names', () => {
    expect(reverseName('192.0.2.10')).toEqual({ input: '192.0.2.10', kind: 'ptr', names: ['10.2.0.192.in-addr.arpa'] })
    expect(reverseName('2001:db8::567:89ab')!.names[0]).toBe('b.a.9.8.7.6.5.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa')
    expect(reverseName('10.0.0.1/32')!.kind).toBe('ptr')
  })

  it('builds aligned zones', () => {
    expect(reverseName('192.0.2.0/24')!.names).toEqual(['2.0.192.in-addr.arpa'])
    expect(reverseName('10.0.0.0/8')!.names).toEqual(['10.in-addr.arpa'])
    expect(reverseName('0.0.0.0/0')!.names).toEqual(['in-addr.arpa'])
    expect(reverseName('2001:db8::/32')!.names).toEqual(['8.b.d.0.1.0.0.2.ip6.arpa'])
  })

  it('lists covering zones for unaligned prefixes', () => {
    expect(reverseName('192.168.4.0/22')!.names).toEqual(['4.168.192.in-addr.arpa', '5.168.192.in-addr.arpa', '6.168.192.in-addr.arpa', '7.168.192.in-addr.arpa'])
    expect(reverseName('2001:db8::/30')!.names).toEqual(['8.b.d.0.1.0.0.2.ip6.arpa', '9.b.d.0.1.0.0.2.ip6.arpa', 'a.b.d.0.1.0.0.2.ip6.arpa', 'b.b.d.0.1.0.0.2.ip6.arpa'])
  })

  it('caps very large zone lists', () => {
    const r = reverseName('10.0.0.0/9')!
    expect(r.names).toHaveLength(128)
    const big = reverseName('2001:db8::/33')!
    expect(big.names).toHaveLength(8)
    expect(reverseName('10.0.0.0/1')!).toMatchObject({ note: undefined })
  })

  it('uses RFC 2317 names beyond /24', () => {
    expect(reverseName('192.0.2.64/26')).toEqual({ input: '192.0.2.64/26', kind: 'classless', names: ['64/26.2.0.192.in-addr.arpa'], note: '2.0.192.in-addr.arpa' })
  })

  it('reports invalid entries', () => {
    expect(reverseNames('192.0.2.1 nope').invalid).toEqual(['nope'])
  })
})
