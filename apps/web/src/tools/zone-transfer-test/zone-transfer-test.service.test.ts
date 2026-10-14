import type { AxfrServer } from './zone-transfer-test.service'
import { describe, expect, it } from 'vitest'
import { serverState, typeSummary } from './zone-transfer-test.service'

describe('zone transfer helpers', () => {
  it('classifies servers', () => {
    expect(serverState({ allowed: true, error: '' } as AxfrServer)).toBe('open')
    expect(serverState({ allowed: false, error: 'refused: dns: bad xfr rcode: 5' } as AxfrServer)).toBe('refused')
    expect(serverState({ allowed: false, error: 'TCP port 53 not reachable' } as AxfrServer)).toBe('error')
  })
  it('summarizes types', () => {
    expect(typeSummary({ NS: 2, A: 10, SOA: 2 })).toBe('A 10, NS 2, SOA 2')
  })
})
