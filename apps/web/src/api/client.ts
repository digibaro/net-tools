import type { components } from './schema'

type Schemas = components['schemas']

export type ApiErrorCode = Schemas['ErrorCode'] | 'NETWORK_ERROR' | 'BAD_RESPONSE'
export type VersionInfo = Schemas['Version']
export type Capabilities = Schemas['Capabilities']
export type EchoInfo = Schemas['Echo']

export class ApiError extends Error {
  constructor(
    public readonly code: ApiErrorCode,
    message: string,
    public readonly status = 0,
    public readonly details: Record<string, unknown> = {},
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export interface RequestOptions {
  signal?: AbortSignal
  fetch?: typeof fetch
}

const API_BASE = '/api'

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null
}

/** GET an enveloped endpoint (docs/API_SPEC.md §1.1); resolves to `data` or throws ApiError. */
export async function apiGet<T>(path: string, options: RequestOptions = {}): Promise<T> {
  return apiRequest<T>(path, { headers: { Accept: 'application/json' } }, options)
}

/** POST JSON to an enveloped endpoint. */
export async function apiPost<T>(path: string, body: unknown, options: RequestOptions = {}): Promise<T> {
  return apiRequest<T>(path, { method: 'POST', headers: { 'Accept': 'application/json', 'Content-Type': 'application/json' }, body: JSON.stringify(body) }, options)
}

async function apiRequest<T>(path: string, init: RequestInit, options: RequestOptions): Promise<T> {
  const doFetch = options.fetch ?? globalThis.fetch
  let res: Response
  try {
    res = await doFetch(`${API_BASE}${path}`, { ...init, signal: options.signal })
  }
  catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError')
      throw err
    throw new ApiError('NETWORK_ERROR', err instanceof Error ? err.message : String(err))
  }

  let body: unknown
  try {
    body = await res.json()
  }
  catch {
    throw new ApiError('BAD_RESPONSE', `Unexpected response (HTTP ${res.status})`, res.status)
  }

  if (isRecord(body) && body.ok === true && 'data' in body)
    return body.data as T

  if (isRecord(body) && body.ok === false && isRecord(body.error)) {
    const e = body.error as Partial<Schemas['ApiError']>
    throw new ApiError(e.code ?? 'INTERNAL', e.message ?? 'Unknown error', res.status, e.details ?? {})
  }
  throw new ApiError('BAD_RESPONSE', `Unexpected response (HTTP ${res.status})`, res.status)
}

export const getVersion = (o?: RequestOptions) => apiGet<VersionInfo>('/v1/version', o)
export const getCapabilities = (o?: RequestOptions) => apiGet<Capabilities>('/v1/capabilities', o)
export const getEcho = (o?: RequestOptions) => apiGet<EchoInfo>('/v1/echo', o)
