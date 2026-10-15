import { describe, expect, it } from 'vitest'
import { analyzeIPv6 } from './ipv6-address-analyzer.service'

describe('ipv6-address-analyzer', () => {
  it('analyzes an EUI-64 link-local address', () => {
    const r = analyzeIPv6('fe80::21b:21ff:fe3a:4b5c%eth0')!
    expect(r.scope).toBe('link')
    expect(r.iid).toMatchObject({ kind: 'eui64', mac: '00:1b:21:3a:4b:5c', macLocal: false })
    expect(r.solicitedNode).toBe('ff02::1:ff3a:4b5c')
    expect(r.prefix64).toBe('fe80::/64')
    expect(r.expanded).toBe('fe80:0000:0000:0000:021b:21ff:fe3a:4b5c')
  })
  it('recognises multicast flags and scope', () => {
    const r = analyzeIPv6('ff3e:30:2001:db8::1')!
    expect(r.multicast).toBe(true)
    expect(r.scope).toBe('global')
    expect(r.flags).toEqual({ transient: true, prefix: true, rp: false })
    expect(r.iid).toBeUndefined()
  })
  it('classifies interface identifiers', () => {
    expect(analyzeIPv6('2001:db8::1')!.iid!.kind).toBe('low')
    expect(analyzeIPv6('2001:db8::5efe:c000:201')!.iid).toMatchObject({ kind: 'isatap', ipv4: '192.0.2.1' })
    expect(analyzeIPv6('2001:db8::3c1a:9b2e:71f0:d4a7')!.iid!.kind).toBe('random')
    expect(analyzeIPv6('2001:db8::/64')!.iid!.kind).toBe('zero')
  })
  it('finds embedded IPv4 and rejects garbage', () => {
    expect(analyzeIPv6('64:ff9b::c000:221')!.embedded[0]).toMatchObject({ kind: 'nat64', ipv4: '192.0.2.33' })
    expect(analyzeIPv6('2001:db8::1')!.globallyReachable).toBe(false)
    expect(analyzeIPv6('not an address')).toBeUndefined()
  })
})
