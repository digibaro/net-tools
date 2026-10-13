import type { TraceHop } from './mtr.service'
import { describe, expect, it } from 'vitest'
import { traceTextReport } from './mtr.service'

describe('mtr report', () => {
  it('renders loss and stats per hop with AS', () => {
    const h: TraceHop = { hop: 1, ip: '192.0.2.1', host: 'gw.example.', sent: 10, recv: 9, loss: 10, lastMs: 2, avgMs: 2.5, bestMs: 1, worstMs: 4, stddevMs: 0.5, rttMs: [], asn: 64500 }
    expect(traceTextReport([h], true)).toContain('gw.example. (192.0.2.1) [AS64500]')
    expect(traceTextReport([h], true)).toContain('loss 10%  snt 10  last 2  avg 2.5  best 1  wrst 4')
  })
})
