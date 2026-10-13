import type { TraceHop } from './traceroute.service'
import { describe, expect, it } from 'vitest'
import { traceTextReport, upsertHop, withGaps } from './traceroute.service'

const hop = (n: number, ip = '', rtt: number[] = []): TraceHop => ({ hop: n, ip, host: '', sent: 3, recv: rtt.length, loss: rtt.length ? 0 : 100, lastMs: rtt.at(-1) ?? 0, avgMs: rtt[0] ?? 0, bestMs: 0, worstMs: 0, stddevMs: 0, rttMs: rtt, asn: null })

describe('traceroute', () => {
  it('upserts and fills gaps', () => {
    let hops = upsertHop([], hop(3, '192.0.2.3', [3]))
    hops = upsertHop(hops, hop(1, '192.0.2.1', [1]))
    hops = upsertHop(hops, hop(1, '192.0.2.1', [1, 2]))
    expect(hops.map(h => h.hop)).toEqual([1, 3])
    expect(withGaps(hops).map(h => h.ip)).toEqual(['192.0.2.1', '', '192.0.2.3'])
  })
  it('renders a traceroute-style report', () => {
    expect(traceTextReport([hop(1, '192.0.2.1', [1.5, 2]), hop(3, '192.0.2.3', [9])], false)).toBe(' 1  192.0.2.1  1.5 ms  2 ms\n 2  *  * * *\n 3  192.0.2.3  9 ms')
  })
})
