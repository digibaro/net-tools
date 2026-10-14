import { describe, expect, it } from 'vitest'
import { buildCurl, emptySpec, parseCurl, quote, tokenize } from './curl-builder.service'

describe('tokenize', () => {
  it('handles quotes, escapes and continuations', () => {
    expect(tokenize(`curl -H 'A: b c' -d "x=\\"1\\"" \\\n  https://e.test/a\\ b $'l1\\nl2'`)).toEqual(['curl', '-H', 'A: b c', '-d', 'x="1"', 'https://e.test/a b', 'l1\nl2'])
    expect(tokenize(`curl 'unterminated`)).toBeUndefined()
  })
})

describe('parseCurl', () => {
  it('parses a typical POST', () => {
    const s = parseCurl(`curl -X POST https://api.test/v1/items -H 'Content-Type: application/json' -H "Authorization: Bearer abc" --data-raw '{"a":1}' -L --compressed -k`)
    if (typeof s === 'string')
      throw new Error(s)
    expect(s.method).toBe('POST')
    expect(s.url).toBe('https://api.test/v1/items')
    expect(s.headers).toEqual([['Content-Type', 'application/json'], ['Authorization', 'Bearer abc']])
    expect(s.data).toBe('{"a":1}')
    expect(s.follow && s.compressed && s.insecure).toBe(true)
  })
  it('infers methods and handles -G, -I, --json, joined flags', () => {
    expect((parseCurl('curl -d a=1 https://e.test') as { method: string }).method).toBe('POST')
    expect(parseCurl('curl -G -d q=x https://e.test/s')).toMatchObject({ method: 'GET', url: 'https://e.test/s?q=x', data: '' })
    expect(parseCurl('curl -I https://e.test')).toMatchObject({ method: 'HEAD' })
    expect(parseCurl(`curl --json '{"x":1}' https://e.test`)).toMatchObject({ method: 'POST', headers: [['Content-Type', 'application/json'], ['Accept', 'application/json']] })
    expect(parseCurl('curl -XPUT --header=X-A:1 https://e.test')).toMatchObject({ method: 'PUT', headers: [['X-A', '1']] })
    expect(parseCurl('curl --tlsv1.3 https://e.test')).toMatchObject({ unknown: ['--tlsv1.3'] })
  })
  it('rejects non-curl input', () => {
    expect(parseCurl('wget https://e.test')).toBe('not-curl')
    expect(parseCurl(`curl 'x`)).toBe('syntax')
  })
})

describe('buildCurl', () => {
  it('round-trips', () => {
    const spec = { ...emptySpec(), url: 'https://e.test/a?b=1&c=2', method: 'PUT', headers: [['X-Name', 'O\'Brien']] as [string, string][], data: 'v=1', follow: true }
    const cmd = buildCurl(spec)
    expect(cmd).toContain(`-H 'X-Name: O'\\''Brien'`)
    expect(parseCurl(cmd)).toMatchObject({ url: spec.url, method: 'PUT', headers: spec.headers, data: 'v=1', follow: true })
  })
  it('quotes only when needed', () => {
    expect(quote('https://e.test/x')).toBe('https://e.test/x')
    expect(quote('a b')).toBe(`'a b'`)
  })
})
