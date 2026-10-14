import type { LatencyRow } from './latency-matrix.service'
import { describe, expect, it } from 'vitest'
import { parseTargets, ranked } from './latency-matrix.service'

const row = (target: string, avg: number, received = 1) => ({ target, stats: { avgMs: avg, received } }) as LatencyRow

describe('latency-matrix', () => {
  it('parses targets', () => {
    expect(parseTargets('a.test, b.test\na.test')).toEqual(['a.test', 'b.test'])
  })
  it('ranks rows', () => {
    expect(ranked([row('slow', 50), row('dead', 0, 0), row('fast', 5)]).map(r => r.target)).toEqual(['fast', 'slow', 'dead'])
  })
})
