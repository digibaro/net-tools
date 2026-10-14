import { describe, expect, it } from 'vitest'
import { convert } from './cidr-range-converter.service'

describe('convert', () => {
  it('cIDR → range', () => {
    expect(convert('10.1.2.3/22')).toEqual({ mode: 'cidr', rows: [{ cidr: '10.1.0.0/22', first: '10.1.0.0', last: '10.1.3.255', addresses: '1,024' }] })
    expect(convert('2001:db8::/126')?.rows[0]).toMatchObject({ first: '2001:db8::', last: '2001:db8::3', addresses: '4' })
  })

  it('range → minimal CIDRs', () => {
    const r = convert('192.168.0.10 - 192.168.0.20')
    expect(r?.mode).toBe('range')
    expect(r?.rows.map(x => x.cidr)).toEqual(['192.168.0.10/31', '192.168.0.12/30', '192.168.0.16/30', '192.168.0.20/32'])
    expect(r && 'total' in r && r.total).toBe('11')
    expect(convert('2001:db8::–2001:db8::ff')?.rows.map(x => x.cidr)).toEqual(['2001:db8::/120'])
  })

  it.each(['10.0.0.5 - 10.0.0.1', '10.0.0.1 - ::1', '10.0.0.1', 'x - y', '', '10.0.0.0/40'])('rejects %j', (s) => {
    expect(convert(s)).toBeUndefined()
  })
})
