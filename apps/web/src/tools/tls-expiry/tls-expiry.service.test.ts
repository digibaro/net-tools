import { describe, expect, it } from 'vitest'
import { parseTargets, sortRows } from './tls-expiry.service'

describe('tls-expiry', () => {
  it('parses targets', () => {
    expect(parseTargets('example.com\nhttps://example.org/path, example.com\n mail.example.com:465 ')).toEqual(['example.com', 'example.org', 'mail.example.com:465'])
  })
  it('sorts by days left', () => {
    const r = (target: string, daysLeft: number | null) => ({ target, daysLeft } as never)
    expect(sortRows([r('a', 90), r('b', null), r('c', 5)]).map((x: { target: string }) => x.target)).toEqual(['c', 'a', 'b'])
  })
})
