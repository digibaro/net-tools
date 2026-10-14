import type { LookingGlass } from './looking-glass.service'
import { describe, expect, it } from 'vitest'
import { pathCounts } from './looking-glass.service'

describe('pathCounts', () => {
  it('counts AS paths', () => {
    const lg = { routes: [{ asPath: '1 2' }, { asPath: '3 2' }, { asPath: '1 2' }] } as LookingGlass
    expect(pathCounts(lg)).toEqual([{ path: '1 2', peers: 2 }, { path: '3 2', peers: 1 }])
  })
})
