import { describe, expect, it } from 'vitest'
import { urgency } from './domain-expiry.service'

describe('urgency', () => {
  it('classifies days left', () => {
    expect(urgency(null)).toBe('unknown')
    expect(urgency(-1)).toBe('expired')
    expect(urgency(30)).toBe('soon')
    expect(urgency(365)).toBe('ok')
  })
})
