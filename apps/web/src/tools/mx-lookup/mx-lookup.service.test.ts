import type { Probe } from './mx-lookup.service'
import { describe, expect, it } from 'vitest'
import { probeStatus } from './mx-lookup.service'

const p = (x: Partial<Probe>): Probe => ({ address: '', connected: true, connectMs: 1, banner: '', greeting: '', extensions: [], starttls: true, tls: null, error: '', ...x })

describe('probeStatus', () => {
  it('classifies probes', () => {
    expect(probeStatus(p({ connected: false, error: 'x' }))).toBe('unreachable')
    expect(probeStatus(p({ starttls: false }))).toBe('no-starttls')
    expect(probeStatus(p({ tls: { verified: true } as Probe['tls'] }))).toBe('ok')
    expect(probeStatus(p({ tls: { verified: false } as Probe['tls'] }))).toBe('cert')
  })
})
