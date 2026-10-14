import type { AsnInfo } from './asn-info.service'
import { describe, expect, it } from 'vitest'
import { neighbourRows } from './asn-info.service'

describe('neighbourRows', () => {
  it('links neighbours', () => {
    const info = { neighbours: [{ asn: 174, type: 'left', power: 10 }] } as AsnInfo
    expect(neighbourRows(info, t => t.toUpperCase())).toEqual([{ asn: 'AS174', link: '/asn-info?asn=AS174', relation: 'LEFT', power: 10 }])
  })
})
