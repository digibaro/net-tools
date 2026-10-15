import { describe, expect, it } from 'vitest'
import { hasPosition, landPath, MAP_H, MAP_W, project } from './remote-map'

describe('remote-map', () => {
  it('projects equirectangular', () => {
    expect(project(0, 0)).toEqual([MAP_W / 2, MAP_H / 2])
    expect(project(-180, 90)).toEqual([0, 0])
    const [x, y] = project(4.9, 52.37) // Amsterdam
    expect(x).toBeCloseTo(493.07, 1)
    expect(y).toBeCloseTo(100.35, 1)
  })
  it('skips probes without a position', () => {
    expect(hasPosition({ latitude: 0, longitude: 0 })).toBe(false)
    expect(hasPosition({ latitude: 52.4, longitude: 4.9 })).toBe(true)
  })
  it('builds the land path', async () => {
    const d = await landPath()
    expect(d.startsWith('M')).toBe(true)
    expect(d.length).toBeGreaterThan(10000)
  })
})
