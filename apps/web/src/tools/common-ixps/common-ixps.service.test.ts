import { describe, expect, it } from 'vitest'
import { ports } from './common-ixps.service'

describe('common-ixps', () => {
  it('summarises ports', () => {
    expect(ports([{ speedMbps: 100000, ipv4: '', ipv6: '', rsPeer: true }, { speedMbps: 500, ipv4: '', ipv6: '', rsPeer: false }])).toBe('100G (RS), 500M')
  })
})
