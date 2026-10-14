import { describe, expect, it } from 'vitest'
import { prefixForCount, split } from './subnet-splitter.service'

describe('split', () => {
  it('splits by new prefix', () => {
    const r = split('192.168.0.0/24', 'prefix', 26)!
    expect(r.rows.map(x => x.cidr)).toEqual(['192.168.0.0/26', '192.168.0.64/26', '192.168.0.128/26', '192.168.0.192/26'])
    expect(r.rows[1]).toMatchObject({ first: '192.168.0.64', last: '192.168.0.127', addresses: '64' })
  })

  it('splits by count, rounding up to a power of two', () => {
    const r = split('10.0.0.0/16', 'count', 5)!
    expect(r.newPrefix).toBe(19)
    expect(r.rows).toHaveLength(8)
  })

  it('normalizes host bits and truncates huge splits', () => {
    expect(split('10.0.0.77/24', 'prefix', 25)!.rows[0]!.cidr).toBe('10.0.0.0/25')
    const r = split('2001:db8::/32', 'prefix', 64)!
    expect(r).toMatchObject({ truncated: true, total: '4,294,967,296' })
    expect(r.rows).toHaveLength(1024)
    expect(r.rows[1]!.cidr).toBe('2001:db8:0:1::/64')
  })

  it.each([
    ['10.0.0.0/24', 'prefix', 23],
    ['10.0.0.0/24', 'prefix', 33],
    ['10.0.0.0', 'prefix', 25],
    ['10.0.0.0/24', 'count', 0],
    ['10.0.0.0/24', 'count', 1000],
  ] as const)('rejects %s %s %i', (cidr, mode, v) => {
    expect(split(cidr, mode, v)).toBeUndefined()
  })

  it('prefixForCount', () => {
    expect(prefixForCount(24, 1)).toBe(24)
    expect(prefixForCount(24, 2)).toBe(25)
    expect(prefixForCount(24, 3)).toBe(26)
  })
})
