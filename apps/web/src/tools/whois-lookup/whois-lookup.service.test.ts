import type { Whois } from './whois-lookup.service'
import { describe, expect, it } from 'vitest'
import { finalResponse } from './whois-lookup.service'

const r = (server: string, text: string, error = '') => ({ server, query: 'x', text, truncated: false, error })

describe('finalResponse', () => {
  it('prefers the last answer with text', () => {
    const w = { chain: [r('iana', 'refer: a'), r('a', 'registry'), r('b', '', 'connection failed')] } as Whois
    expect(finalResponse(w)?.server).toBe('a')
  })
})
