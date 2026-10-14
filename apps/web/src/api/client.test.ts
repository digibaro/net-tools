import { describe, expect, it, vi } from 'vitest'
import { ApiError, apiGet, apiPost, getVersion } from './client'

function fakeFetch(status: number, body: unknown, raw = false): typeof fetch {
  return vi.fn(async () => new Response(raw ? String(body) : JSON.stringify(body), { status }))
}

describe('apiGet', () => {
  it('unwraps a success envelope', async () => {
    const version = { version: '1.0.0', commit: 'abc', buildDate: 'x', goVersion: 'go1.27' }
    const f = fakeFetch(200, { ok: true, data: version })
    await expect(getVersion({ fetch: f })).resolves.toEqual(version)
    expect(f).toHaveBeenCalledWith('/api/v1/version', expect.objectContaining({ headers: { Accept: 'application/json' } }))
  })

  it('posts JSON', async () => {
    const f = fakeFetch(200, { ok: true, data: { hops: [] } })
    await expect(apiPost('/v1/http/request', { url: 'x' }, { fetch: f })).resolves.toEqual({ hops: [] })
    expect(f).toHaveBeenCalledWith('/api/v1/http/request', expect.objectContaining({ method: 'POST', body: '{"url":"x"}' }))
  })

  it('throws ApiError with code, status and details for an error envelope', async () => {
    const f = fakeFetch(400, { ok: false, error: { code: 'INVALID_INPUT', message: 'bad name', details: { field: 'name' } } })
    const err = await apiGet('/v1/x', { fetch: f }).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ code: 'INVALID_INPUT', message: 'bad name', status: 400, details: { field: 'name' } })
  })

  it('maps network failures to NETWORK_ERROR', async () => {
    const f: typeof fetch = vi.fn(async () => {
      throw new TypeError('Failed to fetch')
    })
    await expect(apiGet('/v1/x', { fetch: f })).rejects.toMatchObject({ code: 'NETWORK_ERROR' })
  })

  it('rethrows aborts unchanged', async () => {
    const abort = new DOMException('aborted', 'AbortError')
    const f: typeof fetch = vi.fn(async () => {
      throw abort
    })
    await expect(apiGet('/v1/x', { fetch: f })).rejects.toBe(abort)
  })

  it.each([
    ['non-JSON body', fakeFetch(502, '<html>Bad Gateway</html>', true)],
    ['JSON without envelope', fakeFetch(200, { hello: 'world' })],
  ])('maps %s to BAD_RESPONSE', async (_, f) => {
    await expect(apiGet('/v1/x', { fetch: f })).rejects.toMatchObject({ code: 'BAD_RESPONSE' })
  })
})
