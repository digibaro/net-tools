import { describe, expect, it } from 'vitest'
import { aggregate } from './cidr-aggregator.service'

describe('aggregate', () => {
  it('merges, dedupes and reports invalid lines', () => {
    const r = aggregate(`
      10.0.0.0/24
      10.0.1.0/24, 10.0.1.17
      bogus
      2001:db8::/48 2001:db8:1::/48
      192.168.1.5/24
      300.1.1.1/8
    `)
    expect(r.output).toEqual(['10.0.0.0/23', '192.168.1.0/24', '2001:db8::/47'])
    expect(r.invalid).toEqual(['bogus', '300.1.1.1/8'])
    expect(r.inputCount).toBe(6)
  })

  it('handles empty input', () => {
    expect(aggregate('  ')).toEqual({ output: [], invalid: [], inputCount: 0 })
  })
})
