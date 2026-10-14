import { describe, expect, it } from 'vitest'
import { isPortSpec } from './port-scanner.service'

describe('isPortSpec', () => {
  it.each([['top100', true], ['22,80,8000-8100', true], ['0', false], ['90-80', false], ['x', false], ['70000', false]])('%s → %s', (s, want) => {
    expect(isPortSpec(s)).toBe(want)
  })
})
