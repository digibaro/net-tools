import { afterEach, describe, expect, it, vi } from 'vitest'
import { probe, quality, stats } from './browser-latency.service'

describe('stats', () => {
  it('summarizes samples with loss', () => {
    const s = stats([10, 20, null, 30, 20])!
    expect(s).toEqual({ sent: 5, received: 4, lossPct: 20, min: 10, avg: 20, median: 20, p95: 28.5, max: 30, stddev: 7.1, jitter: 10 })
    expect(stats([])).toBeUndefined()
    expect(stats([null, null])!.lossPct).toBe(100)
  })
  it('rates quality', () => {
    expect(quality(stats([10, 11, 12])!)).toBe('excellent')
    expect(quality(stats([100, 110])!)).toBe('good')
    expect(quality(stats([400, 410])!)).toBe('poor')
  })
})

describe('probe', () => {
  afterEach(() => vi.unstubAllGlobals())
  it('times a request and busts caches', async () => {
    const fetchMock = vi.fn(async (_url: string) => new Response('{}'))
    vi.stubGlobal('fetch', fetchMock)
    let t = 0
    expect(await probe('/api/v1/echo', undefined, () => (t += 7))).toBe(7)
    expect(fetchMock.mock.calls[0]![0]).toMatch(/^\/api\/v1\/echo\?_=/)
  })
  it('returns null on failure', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('', { status: 502 })))
    expect(await probe('/x')).toBeNull()
    vi.stubGlobal('fetch', vi.fn(async () => {
      throw new TypeError('network')
    }))
    expect(await probe('/x')).toBeNull()
  })
})
