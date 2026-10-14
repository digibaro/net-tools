import type { TorCheck } from './tor-exit-check.service'
import { describe, expect, it } from 'vitest'
import { verdict } from './tor-exit-check.service'

describe('verdict', () => {
  it('classifies', () => {
    expect(verdict({ exitNode: true, relays: [] } as unknown as TorCheck)).toBe('exit')
    expect(verdict({ exitNode: false, relays: [{}] } as unknown as TorCheck)).toBe('relay')
    expect(verdict({ exitNode: false, relays: [] } as unknown as TorCheck)).toBe('none')
  })
})
