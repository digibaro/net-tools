import { describe, expect, it } from 'vitest'
import { alertType, isUrlOrDomain, tagType } from './url-reputation.service'

describe('url-reputation', () => {
  it.each([['https://example.com/login', true], ['example.com', true], ['ftp://example.com', false], ['10.0.0.1', false], ['intranet', false], ['', false]])('%s → %s', (s, want) => {
    expect(isUrlOrDomain(s)).toBe(want)
  })
  it('maps verdicts to colours', () => {
    expect(alertType('malicious')).toBe('error')
    expect(alertType('unknown')).toBe('info')
    expect(tagType('skipped')).toBe('default')
  })
})
