import type { RelayResult } from './open-relay-test.service'
import { describe, expect, it } from 'vitest'
import { replyCode, verdict } from './open-relay-test.service'

const base: RelayResult = { address: '', banner: '', startTls: false, open: false, error: '', cases: [{ name: 'external', mailFrom: 'a', rcptTo: 'b', mailReply: '250 Ok', rcptReply: '554 denied', accepted: false }] }

describe('open-relay-test', () => {
  it('derives the verdict', () => {
    expect(verdict(base)).toBe('closed')
    expect(verdict({ ...base, open: true })).toBe('open')
    expect(verdict({ ...base, error: 'connection lost' })).toBe('inconclusive')
    expect(verdict({ ...base, cases: [] })).toBe('inconclusive')
  })
  it('reads reply codes', () => {
    expect(replyCode('554 5.7.1 denied')).toBe(554)
    expect(replyCode('')).toBe(0)
  })
})
