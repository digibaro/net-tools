import type { CloudRanges } from './cloud-ip-lookup.service'
import { describe, expect, it } from 'vitest'
import { failedSources } from './cloud-ip-lookup.service'

describe('failedSources', () => {
  it('lists unavailable lists', () => {
    expect(failedSources({ sources: [{ name: 'A', ranges: 1, error: '' }, { name: 'B', ranges: 0, error: 'x' }] } as CloudRanges)).toEqual(['B'])
  })
})
