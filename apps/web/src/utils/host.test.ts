import { describe, expect, it } from 'vitest'
import { isValidHost } from './host'

describe('isValidHost', () => {
  it.each(['example.com', 'münchen.de', '8.8.8.8', '2001:db8::1', '[2001:db8::1]', 'localhost'])('accepts %s', (s) => {
    expect(isValidHost(s)).toBe(true)
  })
  it.each(['', 'http://x.com', 'a b', '2130706433', '0x7f.1', '127.1', 'user@host'])('rejects %j', (s) => {
    expect(isValidHost(s)).toBe(false)
  })
})
