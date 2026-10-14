import { describe, expect, it } from 'vitest'
import { parseCidr } from '@/utils/ip'
import { acl, asnFilter, isListName, parsePrefixes, prefixList, withUpTo } from './router-snippets.service'

const p = (s: string, ge?: number, le?: number) => ({ cidr: parseCidr(s)!, ge, le })

describe('router-snippets', () => {
  it('parses and deduplicates', () => {
    const r = parsePrefixes('10.0.0.1/24, 10.0.0.0/24\n2001:db8::/32 nope 192.0.2.1')
    expect(r.entries.length).toBe(3)
    expect(r.invalid).toEqual(['nope'])
  })
  it('builds IOS prefix-lists with ranges and v6 split', () => {
    const out = prefixList([p('1.0.0.0/24'), p('5.10.214.0/23', 24, 24), p('10.0.0.0/8', undefined, 24), p('2001:db8::/32')], 'ios', 'AS-X')
    expect(out).toBe([
      'ip prefix-list AS-X seq 5 permit 1.0.0.0/24',
      'ip prefix-list AS-X seq 10 permit 5.10.214.0/23 ge 24 le 24',
      'ip prefix-list AS-X seq 15 permit 10.0.0.0/8 le 24',
      '',
      'ipv6 prefix-list AS-X-V6 seq 5 permit 2001:db8::/32',
    ].join('\n'))
  })
  it('builds Junos, XR, BIRD', () => {
    expect(prefixList([p('10.0.0.0/8', 16, 32)], 'ios', 'L')).toBe('ip prefix-list L seq 5 permit 10.0.0.0/8 ge 16')
    expect(prefixList([p('1.0.0.0/24')], 'junos', 'L')).toBe('set policy-options prefix-list L 1.0.0.0/24')
    expect(prefixList([p('1.0.0.0/24'), p('10.0.0.0/8', undefined, 24)], 'junos', 'L')).toBe('set policy-options route-filter-list L 1.0.0.0/24 exact\nset policy-options route-filter-list L 10.0.0.0/8 upto /24')
    expect(prefixList([p('5.10.214.0/23', 24, 24)], 'junos', 'L')).toContain('prefix-length-range /24-/24')
    expect(prefixList([p('1.0.0.0/24'), p('5.10.214.0/23', 24, 24)], 'iosxr', 'L')).toBe('prefix-set L\n  1.0.0.0/24,\n  5.10.214.0/23 ge 24 le 24\nend-set')
    expect(prefixList([p('1.0.0.0/24'), p('5.10.214.0/23', 24, 24)], 'bird', 'as-x')).toBe('define AS_X = [\n  1.0.0.0/24,\n  5.10.214.0/23{24,24}\n];')
  })
  it('uses EOS syntax', () => {
    expect(prefixList([p('2001:db8::/32')], 'eos', 'L')).toBe('ipv6 prefix-list L\n   seq 10 permit 2001:db8::/32')
    expect(acl([p('10.0.0.0/24')], 'eos', 'A')).toBe('ip access-list A\n   10 permit ip 10.0.0.0/24 any')
  })
  it('builds ACLs with wildcard masks', () => {
    expect(acl([p('10.0.0.0/24'), p('192.0.2.7/32')], 'ios', 'MGMT')).toBe('ip access-list extended MGMT\n permit ip 10.0.0.0 0.0.0.255 any\n permit ip host 192.0.2.7 any')
    expect(acl([p('10.0.0.0/24')], 'junos', 'F', 'deny')).toBe('set firewall family inet filter F term match from source-address 10.0.0.0/24\nset firewall family inet filter F term match then discard')
  })
  it('applies or-longer policy and ASN filters', () => {
    expect(withUpTo([p('10.0.0.0/8'), p('2001:db8::/32')], 24, 48).map(e => e.le)).toEqual([24, 48])
    expect(withUpTo([p('10.0.0.0/25')], 24)[0]!.le).toBeUndefined()
    expect(asnFilter([65000, 65001], 'junos', 'AS-X')).toBe('set policy-options as-list AS_X members [ 65000 65001 ]')
    expect(asnFilter([65000], 'ios', 'X')).toBe('ip as-path access-list X permit _65000$')
    expect(isListName('AS-CUSTOMERS')).toBe(true)
    expect(isListName('bad name')).toBe(false)
  })
})
