import { describe, expect, it } from 'vitest'
import { duration } from './cache-header-analyzer.service'

describe('cache-header-analyzer', () => {
  it('formats lifetimes', () => {
    expect(duration(3600)).toBe('1 h')
    expect(duration(90)).toBe('1 min 30 s')
    expect(duration(31536000)).toBe('365 d')
    expect(duration(0)).toBe('0 s')
    expect(duration(-1)).toBeUndefined()
  })
})
