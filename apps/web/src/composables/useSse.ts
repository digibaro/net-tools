import { onScopeDispose, ref, shallowRef } from 'vue'
import { ApiError } from '@/api/client'
import { buildQuery } from './useApi'

type Params = Record<string, string | number | boolean | undefined>
export type SseHandlers = Record<string, (data: unknown) => void>
export type SseStatus = 'idle' | 'running' | 'done' | 'error' | 'stopped'

export interface EventSourceLike {
  addEventListener: (type: string, listener: (ev: MessageEvent) => void) => void
  close: () => void
  onerror: ((ev: Event) => void) | null
}

function parse(raw: string): unknown {
  try {
    return JSON.parse(raw)
  }
  catch {
    return raw
  }
}

/**
 * Streams a server tool over SSE (docs/API_SPEC.md §4). `done` and `error` events end the stream;
 * stopping closes the EventSource, which cancels the job server-side.
 */
export function useSse(path: string, handlers: SseHandlers, createSource: (url: string) => EventSourceLike = url => new EventSource(url)) {
  const status = ref<SseStatus>('idle')
  const error = shallowRef<ApiError>()
  let source: EventSourceLike | undefined

  function close() {
    source?.close()
    source = undefined
  }

  function start(params: Params = {}) {
    close()
    error.value = undefined
    status.value = 'running'
    const es = createSource(`/api${path}${buildQuery(params)}`)
    source = es
    const events = new Set([...Object.keys(handlers), 'done', 'error'])
    for (const name of events) {
      es.addEventListener(name, (ev: MessageEvent) => {
        if (source !== es)
          return
        const data = parse(ev.data)
        handlers[name]?.(data)
        if (name === 'done') {
          status.value = 'done'
          close()
        }
        else if (name === 'error') {
          const e = (data ?? {}) as { code?: string, message?: string }
          error.value = new ApiError((e.code ?? 'INTERNAL') as ApiError['code'], e.message ?? 'Stream error')
          status.value = 'error'
          close()
        }
      })
    }
    es.onerror = () => {
      if (source !== es)
        return
      error.value = new ApiError('NETWORK_ERROR', 'Connection to the server was lost')
      status.value = 'error'
      close()
    }
  }

  function stop() {
    if (status.value === 'running')
      status.value = 'stopped'
    close()
  }

  onScopeDispose(close)

  return { status, error, start, stop }
}
