import { ref, shallowRef } from 'vue'
import { ApiError, apiGet } from '@/api/client'

type Params = Record<string, string | number | boolean | undefined>

export function buildQuery(params: Params): string {
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '')
      qs.set(k, String(v))
  }
  const s = qs.toString()
  return s ? `?${s}` : ''
}

/**
 * Calls an enveloped GET endpoint; a new run aborts the previous one. `path` excludes `/api`;
 * `suffix` appends path segments (e.g. `/AS13335` for `/v1/asn/{asn}`).
 */
export function useApi<T>(path: string) {
  const data = shallowRef<T>()
  const error = shallowRef<ApiError>()
  const loading = ref(false)
  let controller: AbortController | undefined

  async function run(params: Params = {}, suffix = ''): Promise<T | undefined> {
    controller?.abort()
    const current = new AbortController()
    controller = current
    loading.value = true
    error.value = undefined
    try {
      data.value = await apiGet<T>(`${path}${suffix}${buildQuery(params)}`, { signal: current.signal })
      return data.value
    }
    catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError')
        return undefined
      data.value = undefined
      error.value = err instanceof ApiError ? err : new ApiError('BAD_RESPONSE', String(err))
      return undefined
    }
    finally {
      if (controller === current)
        loading.value = false
    }
  }

  function abort() {
    controller?.abort()
    loading.value = false
  }

  return { data, error, loading, run, abort }
}
