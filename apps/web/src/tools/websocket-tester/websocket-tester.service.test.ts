import type { LogEntry } from './websocket-tester.service'
import { describe, expect, it } from 'vitest'
import { closeKey, hexPreview, isWsUrl, MAX_LOG, parseProtocols, pushLog } from './websocket-tester.service'

describe('websocket-tester', () => {
  it('validates URLs', () => {
    expect(isWsUrl('wss://echo.example/ws')).toBe(true)
    expect(isWsUrl('ws://localhost:8080')).toBe(true)
    expect(isWsUrl('https://example.com')).toBe(false)
    expect(isWsUrl('nope')).toBe(false)
  })
  it('parses subprotocols and close codes', () => {
    expect(parseProtocols('graphql-ws, mqtt ,')).toEqual(['graphql-ws', 'mqtt'])
    expect(closeKey(1006)).toBe('abnormal')
    expect(closeKey(4001)).toBe('other')
  })
  it('previews binary frames and caps the log', () => {
    expect(hexPreview(new Uint8Array([0, 255, 16]).buffer)).toBe('00 ff 10')
    let log: LogEntry[] = Array.from({ length: MAX_LOG }, (_, i) => ({ dir: 'in' as const, at: i, text: '', bytes: 0 }))
    log = pushLog(log, { dir: 'out', at: 999, text: 'x', bytes: 1 })
    expect(log).toHaveLength(MAX_LOG)
    expect(log.at(-1)!.at).toBe(999)
  })
})
