import { describe, expect, it } from 'vitest'
import { decode, encode, parseUrl } from './url-parser.service'

describe('parseUrl', () => {
  it('splits all components and query params', () => {
    expect(parseUrl('https://alice:s3cret@Example.com:8443/a/b%20c?x=1&y=two%20words&x=3#frag')).toEqual({
      href: 'https://alice:s3cret@example.com:8443/a/b%20c?x=1&y=two%20words&x=3#frag',
      origin: 'https://example.com:8443',
      protocol: 'https:',
      username: 'alice',
      hasPassword: true,
      hostname: 'example.com',
      port: '8443',
      effectivePort: '8443',
      pathname: '/a/b%20c',
      search: '?x=1&y=two%20words&x=3',
      hash: '#frag',
      params: [{ key: 'x', value: '1' }, { key: 'y', value: 'two words' }, { key: 'x', value: '3' }],
    })
  })

  it('fills in default ports and rejects non-URLs', () => {
    expect(parseUrl('http://example.com')?.effectivePort).toBe('80')
    expect(parseUrl('example.com')).toBeUndefined()
    expect(parseUrl('')).toBeUndefined()
  })
})

describe('encode / decode', () => {
  it('round-trips components', () => {
    const s = 'a b&c=d/é?'
    expect(encode(s, 'component')).toBe('a%20b%26c%3Dd%2F%C3%A9%3F')
    expect(decode(encode(s, 'component'), 'component')).toBe(s)
    expect(encode('https://x.com/a b?q=é', 'uri')).toBe('https://x.com/a%20b?q=%C3%A9')
  })

  it('treats + as space and rejects malformed escapes', () => {
    expect(decode('a+b%21', 'component')).toBe('a b!')
    expect(decode('%E0%A4%A', 'component')).toBeUndefined()
  })
})
