import { describe, expect, it } from 'vitest'
import { splitNames } from './ct-search.service'

describe('splitNames', () => {
  it('separates wildcards', () => {
    expect(splitNames(['*.example.com', 'example.com', 'www.example.com'])).toEqual({ hosts: ['example.com', 'www.example.com'], wildcards: ['*.example.com'] })
  })
})
