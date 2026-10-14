import { describe, expect, it } from 'vitest'
import { saving } from './compression-check.service'

describe('saving', () => {
  it('formats savings', () => {
    expect(saving(0.18)).toBe('−82 %')
    expect(saving(1)).toBe('')
    expect(saving(0)).toBe('')
  })
})
