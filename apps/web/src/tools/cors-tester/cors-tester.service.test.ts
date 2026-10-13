import { describe, expect, it } from 'vitest'
import { example } from './cors-tester.service'

describe('cors-tester', () => {
  it('works', () => {
    expect(example(' x ')).toBe('x')
  })
})
