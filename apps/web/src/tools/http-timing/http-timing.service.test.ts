import { describe, expect, it } from 'vitest'
import { phases, ttfbRating } from './http-timing.service'

describe('http-timing', () => {
  it('derives sequential phases', () => {
    const p = phases({ resolveMs: 5, connectMs: 10, tlsMs: 20, ttfbMs: 100, transferMs: 15, totalMs: 155 })
    expect(p.map(x => [x.id, x.ms, x.startMs])).toEqual([['resolve', 5, 0], ['connect', 10, 5], ['tls', 20, 15], ['ttfb', 70, 35], ['transfer', 15, 105]])
  })
  it('rates TTFB', () => {
    expect(ttfbRating(50)).toBe('fast')
    expect(ttfbRating(500)).toBe('ok')
    expect(ttfbRating(1500)).toBe('slow')
  })
})
