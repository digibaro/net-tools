import type { EventSourceLike } from './useSse'
import { describe, expect, it, vi } from 'vitest'
import { effectScope, ref } from 'vue'
import { buildQuery } from './useApi'
import { parseQueryValue } from './useQueryParam'
import { useSse } from './useSse'
import { useValidation } from './useValidation'

describe('parseQueryValue', () => {
  it.each([
    [undefined, 'x', 'x'],
    ['abc', 'x', 'abc'],
    [['first', 'second'], 'x', 'first'],
    ['42', 0, 42],
    ['4.5', 0, 4.5],
    ['nope', 7, 7],
    ['', 7, 7],
    ['true', false, true],
    ['false', true, false],
    ['yes', true, true],
  ])('parses %j with default %j', (raw, def, want) => {
    expect(parseQueryValue(raw, def)).toBe(want)
  })
})

describe('buildQuery', () => {
  it('drops empty values and encodes', () => {
    expect(buildQuery({ a: 'x y', b: undefined, c: '', d: 3, e: false })).toBe('?a=x+y&d=3&e=false')
    expect(buildQuery({})).toBe('')
  })
})

describe('useValidation', () => {
  const rules = [{ validator: (v: string) => v.length >= 3, message: 'too short' }]

  it('treats empty input as valid unless required', () => {
    expect(useValidation({ source: '', rules }).value.isValid).toBe(true)
    expect(useValidation({ source: '', rules, required: true }).value).toMatchObject({ isValid: false, message: 'too short' })
  })

  it('reacts to source changes and catches throwing validators', () => {
    const source = ref('ab')
    const v = useValidation({ source, rules: [...rules, { validator: () => {
      throw new Error('x')
    }, message: 'boom' }] })
    expect(v.value).toMatchObject({ isValid: false, message: 'too short', status: 'error' })
    source.value = 'abcd'
    expect(v.value).toMatchObject({ isValid: false, message: 'boom' })
  })
})

class FakeSource implements EventSourceLike {
  listeners = new Map<string, (ev: MessageEvent) => void>()
  closed = false
  onerror: ((ev: Event) => void) | null = null
  constructor(public url: string) {}
  addEventListener(type: string, l: (ev: MessageEvent) => void) {
    this.listeners.set(type, l)
  }

  emit(type: string, data: unknown) {
    this.listeners.get(type)?.({ data: JSON.stringify(data) } as MessageEvent)
  }

  close() {
    this.closed = true
  }
}

describe('useSse', () => {
  function setup() {
    const sources: FakeSource[] = []
    const onReply = vi.fn()
    const scope = effectScope()
    const sse = scope.run(() => useSse('/v1/ping/stream', { reply: onReply }, (url) => {
      const s = new FakeSource(url)
      sources.push(s)
      return s
    }))!
    return { sse, sources, onReply, scope }
  }

  it('dispatches typed events and finishes on done', () => {
    const { sse, sources, onReply } = setup()
    sse.start({ target: 'example.com', count: 3 })
    expect(sources[0]!.url).toBe('/api/v1/ping/stream?target=example.com&count=3')
    expect(sse.status.value).toBe('running')
    sources[0]!.emit('reply', { seq: 1, rttMs: 3.2 })
    expect(onReply).toHaveBeenCalledWith({ seq: 1, rttMs: 3.2 })
    sources[0]!.emit('done', { summary: {} })
    expect(sse.status.value).toBe('done')
    expect(sources[0]!.closed).toBe(true)
  })

  it('maps error events to ApiError', () => {
    const { sse, sources } = setup()
    sse.start()
    sources[0]!.emit('error', { code: 'TARGET_PRIVATE_DENIED', message: 'private' })
    expect(sse.status.value).toBe('error')
    expect(sse.error.value).toMatchObject({ code: 'TARGET_PRIVATE_DENIED', message: 'private' })
  })

  it('reports connection loss and ignores events from a replaced stream', () => {
    const { sse, sources, onReply } = setup()
    sse.start()
    sse.start()
    expect(sources[0]!.closed).toBe(true)
    sources[0]!.emit('reply', { seq: 1 })
    expect(onReply).not.toHaveBeenCalled()
    sources[1]!.onerror?.(new Event('error'))
    expect(sse.error.value?.code).toBe('NETWORK_ERROR')
  })

  it('stop closes the stream and scope disposal cleans up', () => {
    const { sse, sources, scope } = setup()
    sse.start()
    sse.stop()
    expect(sse.status.value).toBe('stopped')
    sse.start()
    scope.stop()
    expect(sources[1]!.closed).toBe(true)
  })
})
