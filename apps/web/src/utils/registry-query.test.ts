import { describe, expect, it } from 'vitest'
import { isRegistryQuery } from './registry-query'

describe('isRegistryQuery', () => {
  it.each([
    ['example.com', true],
    ['8.8.8.8', true],
    ['2001:db8::1', true],
    ['AS13335', true],
    ['13335', true],
    ['localhost', false],
    ['bad host', false],
    ['', false],
  ])('%s → %s', (q, want) => {
    expect(isRegistryQuery(q)).toBe(want)
  })
})
