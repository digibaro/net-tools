import { describe, expect, it } from 'vitest'
import { isValidHttpUrl } from './url'

describe('isValidHttpUrl', () => {
  it.each(['example.com', 'https://example.com/a?b', 'http://[2001:db8::1]:8080/'])('accepts %s', (s) => {
    expect(isValidHttpUrl(s)).toBe(true)
  })
  it.each(['', 'ftp://x.com', 'https://user:pw@x.com', 'http://'])('rejects %j', (s) => {
    expect(isValidHttpUrl(s)).toBe(false)
  })
})
