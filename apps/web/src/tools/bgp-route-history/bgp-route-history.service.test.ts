import type { RouteHistory } from './bgp-route-history.service'
import { describe, expect, it } from 'vitest'
import { originVerdict, visibility } from './bgp-route-history.service'

describe('bgp-route-history', () => {
  it('computes visibility and origin verdict', () => {
    const h = { seeing: 39, fullTablePeers: 40, origins: [{ current: true }, { current: false }] } as RouteHistory
    expect(visibility(h)).toBe(97.5)
    expect(originVerdict(h)).toBe('single')
    expect(originVerdict({ ...h, origins: [{ current: true }, { current: true }] } as RouteHistory)).toBe('moas')
    expect(visibility({ ...h, fullTablePeers: 0 })).toBeUndefined()
  })
})
