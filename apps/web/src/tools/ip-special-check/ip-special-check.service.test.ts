import { describe, expect, it } from 'vitest'
import { checkIps } from './ip-special-check.service'

describe('checkIps', () => {
  it('classifies a mixed list', () => {
    const r = checkIps('10.0.0.1 8.8.8.8 ::1 2606:4700::1111 4000::1 nope')
    expect(r.rows.map(x => [x.ip, x.name, x.bogon])).toEqual([
      ['10.0.0.1', 'Private-Use', true],
      ['8.8.8.8', 'Public', false],
      ['::1', 'Loopback Address', true],
      ['2606:4700::1111', 'Global Unicast', false],
      ['4000::1', 'Unallocated / reserved by IETF', true],
    ])
    expect(r.rows[0]).toMatchObject({ range: '10.0.0.0/8', rfc: 'RFC 1918' })
    expect(r.invalid).toEqual(['nope'])
  })
})
