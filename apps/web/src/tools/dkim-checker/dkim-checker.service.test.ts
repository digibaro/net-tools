import { describe, expect, it } from 'vitest'
import { isValidSelector } from './dkim-checker.service'

describe('isValidSelector', () => {
  it('validates selectors', () => {
    expect(isValidSelector('selector1')).toBe(true)
    expect(isValidSelector('s1.sub')).toBe(true)
    expect(isValidSelector('bad selector')).toBe(false)
    expect(isValidSelector('')).toBe(false)
  })
})
