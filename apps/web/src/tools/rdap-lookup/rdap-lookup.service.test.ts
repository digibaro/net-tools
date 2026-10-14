import { describe, expect, it } from 'vitest'
import { day } from './rdap-lookup.service'

describe('day', () => {
  it('formats dates', () => {
    expect(day('2030-08-13T04:00:00Z')).toBe('2030-08-13')
    expect(day('garbage')).toBe('garbage')
  })
})
