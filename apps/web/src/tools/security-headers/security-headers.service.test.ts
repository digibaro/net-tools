import { describe, expect, it } from 'vitest'
import { example } from './security-headers.service'

describe('security-headers', () => {
  it('works', () => {
    expect(example(' x ')).toBe('x')
  })
})
