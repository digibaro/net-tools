import { describe, expect, it } from 'vitest'
import { isCountryCode, render } from './country-ip-blocks.service'

const list = ['192.0.2.0/24', '2001:db8::/32']

describe('country-ip-blocks', () => {
  it('validates country codes', () => {
    expect(isCountryCode('nl')).toBe(true)
    expect(isCountryCode('NLD')).toBe(false)
  })
  it('renders config formats', () => {
    expect(render(list, 'plain', 'NL')).toBe('192.0.2.0/24\n2001:db8::/32')
    expect(render(list, 'cisco', 'NL')).toBe('ip prefix-list NL permit 192.0.2.0/24\nipv6 prefix-list NL-V6 permit 2001:db8::/32')
    expect(render(list, 'juniper', 'NL')).toContain('        2001:db8::/32;')
    expect(render(list, 'nginx', 'NL').split('\n').at(-1)).toBe('allow all;')
    expect(render(list, 'ipset', 'NL')).toContain('create NL-v6 hash:net family inet6')
    expect(render(list, 'mikrotik', 'NL')).toBe('/ip firewall address-list\nadd list=NL address=192.0.2.0/24\n/ipv6 firewall address-list\nadd list=NL address=2001:db8::/32')
  })
})
