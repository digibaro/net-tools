import { describe, expect, it } from 'vitest'
import { drain, mbps, nextUploadSize, randomPayload } from './speed-test.service'

describe('speed-test', () => {
  it('computes Mbit/s', () => {
    expect(mbps(12_500_000, 1000)).toBe(100)
    expect(mbps(1, 0)).toBe(0)
  })
  it('makes random payloads', () => {
    const p = randomPayload(200_000)
    expect(p.length).toBe(200_000)
    expect(new Set(p.subarray(0, 1000)).size).toBeGreaterThan(100)
  })
  it('counts streamed bytes', async () => {
    let n = 0
    await drain(new Response(new Uint8Array(70_000)), b => (n += b))
    expect(n).toBe(70_000)
  })
})

describe('nextUploadSize', () => {
  it('doubles fast requests up to the server limit and keeps slow ones', () => {
    expect(nextUploadSize(8 << 20, 100)).toBe(16 << 20)
    expect(nextUploadSize(32 << 20, 50)).toBe(32 << 20)
    expect(nextUploadSize(8 << 20, 900)).toBe(8 << 20)
  })
})
