import { describe, expect, it } from 'vitest'
import { example } from './dns-record-types.service'

describe('dns-record-types', () => {
  it('works', () => {
    expect(example(' x ')).toBe('x')
  })
})
