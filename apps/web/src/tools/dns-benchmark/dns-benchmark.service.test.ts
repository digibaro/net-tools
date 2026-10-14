import type { BenchResolver } from './dns-benchmark.service'
import { describe, expect, it } from 'vitest'
import { liveOrder, successPct } from './dns-benchmark.service'

const r = (name: string, answered: number, medianMs: number) => ({ name, answered, medianMs, queries: 4 }) as BenchResolver

describe('dns-benchmark', () => {
  it('orders results', () => {
    expect(liveOrder([r('dead', 0, 0), r('slow', 4, 30), r('fast', 4, 5)]).map(x => x.name)).toEqual(['fast', 'slow', 'dead'])
  })
  it('computes success', () => {
    expect(successPct(r('x', 3, 1))).toBe(75)
  })
})
