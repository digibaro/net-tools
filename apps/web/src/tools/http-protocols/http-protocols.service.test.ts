import type { HttpProtocols } from './http-protocols.service'
import { describe, expect, it } from 'vitest'
import { best, isHttpsUrl } from './http-protocols.service'

describe('http-protocols', () => {
  it('picks the best protocol', () => {
    expect(best({ http2: true, http3: { ok: true } } as HttpProtocols)).toBe('HTTP/3')
    expect(best({ http2: true, http3: { ok: false } } as HttpProtocols)).toBe('HTTP/2')
    expect(best({ http2: false, http3: { ok: false } } as HttpProtocols)).toBe('HTTP/1.1')
  })
  it('requires https', () => {
    expect(isHttpsUrl('https://example.com')).toBe(true)
    expect(isHttpsUrl('http://example.com')).toBe(false)
  })
})
