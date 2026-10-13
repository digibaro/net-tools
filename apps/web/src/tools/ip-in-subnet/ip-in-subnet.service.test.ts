import { describe, expect, it } from 'vitest'
import { matchIps } from './ip-in-subnet.service'

describe('matchIps', () => {
  it('matches each IP against all prefixes, most specific first', () => {
    const r = matchIps('10.1.2.3\n192.0.2.1\n2001:db8::5\nfoo', '10.0.0.0/8, 10.1.0.0/16 2001:db8::/32 bar 10.1.2.3')
    expect(r.rows).toEqual([
      { ip: '10.1.2.3', matched: true, prefixes: ['10.1.2.3/32', '10.1.0.0/16', '10.0.0.0/8'] },
      { ip: '192.0.2.1', matched: false, prefixes: [] },
      { ip: '2001:db8::5', matched: true, prefixes: ['2001:db8::/32'] },
    ])
    expect(r.invalidIps).toEqual(['foo'])
    expect(r.invalidPrefixes).toEqual(['bar'])
  })

  it('never matches across families', () => {
    expect(matchIps('::ffff:10.0.0.1', '10.0.0.0/8').rows[0]!.matched).toBe(false)
  })
})
