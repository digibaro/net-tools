import { describe, expect, it } from 'vitest'
import { clampToLimits } from './tcp-ping.service'

describe('clampToLimits', () => {
  it('respects server limits', () => {
    expect(clampToLimits(50, 50, { pingMaxCount: 20, pingMinIntervalMs: 200 })).toEqual({ count: 20, interval: 200 })
    expect(clampToLimits(0, 9000)).toEqual({ count: 1, interval: 5000 })
  })
})
