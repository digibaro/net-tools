import { describe, expect, it } from 'vitest'
import { bdp, mathisThroughput, windowThroughput } from './bdp-calculator.service'

describe('bdp', () => {
  it('computes the bandwidth-delay product and window scale', () => {
    // 1 Gbit/s × 50 ms = 50 Mbit = 6.25 MB → 65535·2^7 = 8.39 MB ≥ 6.25 MB
    expect(bdp(1e9, 50)).toEqual({ bdpBits: 5e7, bdpBytes: 6.25e6, windowScale: 7 })
    expect(bdp(10e6, 10)?.windowScale).toBe(0)
    expect(bdp(1e15, 1000)?.windowScale).toBeNull()
    expect(bdp(0, 10)).toBeUndefined()
  })
})

describe('windowThroughput', () => {
  it('caps throughput by window / RTT', () => {
    // 64 KiB window, 100 ms RTT → 5.24 Mbit/s
    expect(windowThroughput(65_535, 100)).toBeCloseTo(5_242_800)
    expect(windowThroughput(0, 10)).toBeUndefined()
  })
})

describe('mathisThroughput', () => {
  it('bounds throughput under loss', () => {
    // MSS 1460, 100 ms, 0.1 % loss → 1460*8/0.1 * 1.22/0.0316 ≈ 4.51 Mbit/s
    expect(mathisThroughput(1460, 100, 0.1)! / 1e6).toBeCloseTo(4.506, 2)
    expect(mathisThroughput(1460, 100, 0)).toBeUndefined()
  })
})
