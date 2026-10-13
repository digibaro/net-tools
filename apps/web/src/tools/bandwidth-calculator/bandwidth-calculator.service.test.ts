import { describe, expect, it } from 'vitest'
import { efficiency, requiredRate, transferTime } from './bandwidth-calculator.service'

describe('bandwidth calculator', () => {
  it('computes raw transfer time', () => {
    // 1 GB over 100 Mbit/s = 80 s
    expect(transferTime(8e9, 100e6, 1)).toBe(80)
  })

  it('applies protocol overhead', () => {
    expect(efficiency('tcp-ipv4')).toBeCloseTo(0.9493, 4)
    expect(transferTime(8e9, 100e6, efficiency('tcp-ipv4'))).toBeCloseTo(84.27, 2)
    expect(efficiency('custom', 10)).toBeCloseTo(0.9)
    expect(efficiency('custom', 150)).toBeGreaterThan(0)
  })

  it('computes the required rate', () => {
    expect(requiredRate(8e9, 80, 1)).toBe(100e6)
  })

  it('rejects impossible input', () => {
    expect(transferTime(8, 0, 1)).toBeUndefined()
    expect(requiredRate(8, 0, 1)).toBeUndefined()
    expect(transferTime(Number.NaN, 1, 1)).toBeUndefined()
  })
})
