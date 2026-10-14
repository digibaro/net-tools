import { describe, expect, it } from 'vitest'
import { agreement, groupLabel } from './dns-propagation.service'

describe('dns-propagation', () => {
  it('labels groups', () => {
    expect(groupLabel(0)).toBe('A')
    expect(groupLabel(2)).toBe('C')
    expect(groupLabel(-1)).toBe('—')
  })
  it('computes agreement', () => {
    expect(agreement({ groups: [{ answers: [], rcode: 'NOERROR', count: 3 }, { answers: [], rcode: 'NOERROR', count: 1 }] })).toBe(75)
    expect(agreement({ groups: [] })).toBe(0)
  })
})
