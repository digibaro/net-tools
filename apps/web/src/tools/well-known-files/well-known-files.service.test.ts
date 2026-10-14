import type { WellKnownFile } from './well-known-files.service'
import { describe, expect, it } from 'vitest'
import { fileState } from './well-known-files.service'

describe('fileState', () => {
  it('classifies fetches', () => {
    expect(fileState({ found: true, status: 200 } as WellKnownFile)).toBe('found')
    expect(fileState({ found: false, status: 404 } as WellKnownFile)).toBe('missing')
    expect(fileState({ found: false, status: 500 } as WellKnownFile)).toBe('error')
  })
})
