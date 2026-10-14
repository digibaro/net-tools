import type { DnsWildcard } from './wildcard-detector.service'
import { describe, expect, it } from 'vitest'
import { wildcardAnswers } from './wildcard-detector.service'

describe('wildcardAnswers', () => {
  it('deduplicates answers', () => {
    const a = { name: 'x.example.com.', type: 'A', ttl: 300, class: 'IN', data: '192.0.2.1' }
    const r = { domain: 'example.com', probes: [{ name: 'x', rcode: 'NOERROR', answers: [a] }, { name: 'y', rcode: 'NOERROR', answers: [a] }] } as DnsWildcard
    expect(wildcardAnswers(r)).toEqual(['*.example.com. 300 A 192.0.2.1'])
  })
})
