import type { VlsmResult } from './vlsm-planner.service'
import { describe, expect, it } from 'vitest'
import { parseRequirements, plan, prefixFor } from './vlsm-planner.service'

describe('parseRequirements', () => {
  it('accepts names and bare counts', () => {
    expect(parseRequirements('LAN 50\nWiFi: 100\n\n20')).toEqual([{ name: 'LAN', hosts: 50 }, { name: 'WiFi', hosts: 100 }, { name: 'Subnet 3', hosts: 20 }])
    expect(parseRequirements('Office floor 2 = 12')).toEqual([{ name: 'Office floor 2', hosts: 12 }])
    expect(parseRequirements('LAN fifty')).toBeUndefined()
    expect(parseRequirements('0')).toBeUndefined()
    expect(parseRequirements('')).toBeUndefined()
  })
})

describe('prefixFor', () => {
  it.each([
    [4, 50, 26],
    [4, 62, 26],
    [4, 63, 25],
    [4, 2, 31],
    [4, 1, 32],
    [6, 256, 120],
  ] as const)('v%d %d hosts → /%d', (v, hosts, want) => {
    expect(prefixFor(v, hosts, true)).toBe(want)
  })
  it('uses /30 for point-to-point links when RFC 3021 is off', () => {
    expect(prefixFor(4, 2, false)).toBe(30)
  })
})

describe('plan', () => {
  it('allocates largest first, aligned', () => {
    const r = plan('192.168.1.0/24', 'A 50\nB 100\nC 20\nD 2') as VlsmResult
    expect(r.allocations.map(a => `${a.name} ${a.cidr}`)).toEqual(['B 192.168.1.0/25', 'A 192.168.1.128/26', 'C 192.168.1.192/27', 'D 192.168.1.224/31'])
    expect(r.allocations[0]).toMatchObject({ first: '192.168.1.1', last: '192.168.1.126', usable: '126' })
    expect(r.free).toEqual(['192.168.1.226/31', '192.168.1.228/30', '192.168.1.232/29', '192.168.1.240/28'])
    expect(r.used).toBe(226n)
  })
  it('reports errors', () => {
    expect(plan('10.0.0.0/30', '10')).toBe('no-fit')
    expect(plan('192.168.1.0/24', '200\n100')).toBe('no-fit')
    expect(plan('nope', '10')).toBe('network')
    expect(plan('10.0.0.0/8', 'x')).toBe('requirements')
  })
  it('plans IPv6', () => {
    const r = plan('2001:db8::/48', 'a 65536\nb 10') as VlsmResult
    expect(r.allocations.map(a => a.cidr)).toEqual(['2001:db8::/112', '2001:db8::1:0/124'])
  })
})
