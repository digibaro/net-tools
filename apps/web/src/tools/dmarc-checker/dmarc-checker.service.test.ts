import { describe, expect, it } from 'vitest'
import { strength } from './dmarc-checker.service'

describe('strength', () => {
  it('rates policies', () => {
    expect(strength('reject', 100)).toBe('strong')
    expect(strength('quarantine', 100)).toBe('strong')
    expect(strength('quarantine', 50)).toBe('weak')
    expect(strength('none', 100)).toBe('none')
  })
})
