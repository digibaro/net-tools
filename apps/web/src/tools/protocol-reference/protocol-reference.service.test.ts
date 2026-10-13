import { describe, expect, it } from 'vitest'
import { example } from './protocol-reference.service'

describe('protocol-reference', () => {
  it('works', () => {
    expect(example(' x ')).toBe('x')
  })
})
