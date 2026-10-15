import type { PlanResult } from './ipam-sheet.service'
import { describe, expect, it } from 'vitest'
import { buildPlan, sizeToPrefix } from './ipam-sheet.service'

describe('ipam-sheet', () => {
  it('reads sizes as prefix or hosts', () => {
    expect(sizeToPrefix(4, '/26')).toBe(26)
    expect(sizeToPrefix(4, '24')).toBe(24)
    expect(sizeToPrefix(4, '200')).toBe(24)
    expect(sizeToPrefix(4, 'h20')).toBe(27)
    expect(sizeToPrefix(6, '/64')).toBe(64)
    expect(sizeToPrefix(4, 'big')).toBeUndefined()
  })
  it('places fixed networks first and fills around them in order', () => {
    const r = buildPlan('10.10.0.0/22', 'Mgmt, 99, /28, 10.10.0.0/28\nServers, 10, /24\nUsers, 20, 200 hosts, , floor 2\nP2P, , /31') as PlanResult
    expect(r.rows.map(x => x.cidr)).toEqual(['10.10.0.0/28', '10.10.1.0/24', '10.10.2.0/24', '10.10.0.16/31'])
    expect(r.rows[1]).toMatchObject({ gateway: '10.10.1.1', mask: '255.255.255.0', usable: '254', vlan: '10' })
    expect(r.rows[2]!.note).toBe('floor 2')
    expect(r.free).toContain('10.10.3.0/24')
    expect(r.used).toBe(16n + 256n + 256n + 2n)
  })
  it('reports per-row errors', () => {
    const r = buildPlan('192.168.0.0/24', 'A, 1, /25\nB, 2, /25\nC, 3, /25\nD, 4, /24, 10.0.0.0/24\nE, 5, /28, 192.168.0.0/28\nF, 6, huge') as PlanResult
    // E is fixed, so it is placed first: A gets the upper half, B and C find no room.
    expect(r.rows.map(x => x.error)).toEqual([undefined, 'no-space', 'no-space', 'outside', undefined, 'size'])
    expect(r.rows[0]!.cidr).toBe('192.168.0.128/25')
    expect(r.free).toEqual(['192.168.0.16/28', '192.168.0.32/27', '192.168.0.64/26'])
    const o = buildPlan('192.168.0.0/24', 'A, 1, /26, 192.168.0.0/26\nB, 2, /27, 192.168.0.32/27') as PlanResult
    expect(o.rows[1]!.error).toBe('overlap')
  })
  it('plans IPv6', () => {
    const r = buildPlan('2001:db8:100::/56', 'LAN, 10, /64\nGuest, 20, /64\nVPN, , /60') as PlanResult
    expect(r.rows.map(x => x.cidr)).toEqual(['2001:db8:100::/64', '2001:db8:100:1::/64', '2001:db8:100:10::/60'])
    expect(r.rows[0]!.gateway).toBe('2001:db8:100::1')
    expect(buildPlan('nonsense', '')).toBe('block')
  })
})
