import { describe, expect, it } from 'vitest'
import { example } from './http-timing.service'

describe('http-timing', () => {
  it('works', () => {
    expect(example(' x ')).toBe('x')
  })
})
