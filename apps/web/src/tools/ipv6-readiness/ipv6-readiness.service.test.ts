import type { Ipv6Host } from './ipv6-readiness.service'
import { describe, expect, it } from 'vitest'
import { hostState } from './ipv6-readiness.service'

const h = (aaaa: string[], reachable: boolean | null) => ({ aaaa, reachable }) as Ipv6Host

describe('hostState', () => {
  it('classifies hosts', () => {
    expect(hostState(h([], null))).toBe('no-aaaa')
    expect(hostState(h(['2001:db8::1'], null))).toBe('untested')
    expect(hostState(h(['2001:db8::1'], true))).toBe('ok')
    expect(hostState(h(['2001:db8::1'], false))).toBe('unreachable')
  })
})
