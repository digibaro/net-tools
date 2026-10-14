import { describe, expect, it } from 'vitest'
import { ipv4CheatSheet, ipv6CheatSheet } from './subnet-cheat-sheet.service'

describe('subnet cheat sheet', () => {
  it('covers /0 to /32', () => {
    const rows = ipv4CheatSheet()
    expect(rows).toHaveLength(33)
    expect(rows[24]).toEqual({ prefix: '/24', netmask: '255.255.255.0', wildcard: '0.0.0.255', addresses: '256', usable: '254', size: '1 × /24' })
    expect(rows[30]).toMatchObject({ usable: '2', size: '1/64 of a /24' })
    expect(rows[31]).toMatchObject({ usable: '2' })
    expect(rows[0]).toMatchObject({ netmask: '0.0.0.0', addresses: '4,294,967,296', size: '16,777,216 × /24' })
  })

  it('lists common IPv6 prefixes', () => {
    const rows = ipv6CheatSheet()
    expect(rows.find(r => r.prefix === '/48')?.subnets64).toBe('65,536')
    expect(rows.find(r => r.prefix === '/64')?.subnets64).toBe('1')
    expect(rows.find(r => r.prefix === '/127')?.subnets64).toBe('—')
  })
})
