import { describe, expect, it } from 'vitest'
import { daysUntil, isValidEndpoint } from './encrypted-dns-tester.service'

describe('encrypted-dns-tester', () => {
  it.each([
    ['cloudflare', 'dot', true],
    ['dns.google', 'dot', true],
    ['1.1.1.1:853', 'dot', true],
    ['[2606:4700:4700::1111]:853', 'dot', true],
    ['https://dns.google/dns-query', 'doh', true],
    ['http://dns.google/dns-query', 'doh', false],
    ['bad host', 'dot', false],
    ['', 'dot', false],
  ] as const)('%s (%s) → %s', (v, tr, want) => {
    expect(isValidEndpoint(v, tr)).toBe(want)
  })
  it('counts days', () => {
    expect(daysUntil('2026-01-11T00:00:00Z', Date.parse('2026-01-01T00:00:00Z'))).toBe(10)
  })
})
