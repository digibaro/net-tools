import { describe, expect, it } from 'vitest'
import { isValidHeaderList, isValidOrigin } from './cors-tester.service'

describe('cors-tester', () => {
  it.each(['https://app.example.com', 'http://localhost:3000', 'https://app.example.com/'])('origin %s ok', (s) => {
    expect(isValidOrigin(s)).toBe(true)
  })
  it.each(['app.example.com', 'https://app.example.com/path', 'https://a.com?x=1', 'ftp://a.com', ''])('origin %j rejected', (s) => {
    expect(isValidOrigin(s)).toBe(false)
  })
  it('validates header lists', () => {
    expect(isValidHeaderList('Authorization, X-Requested-With')).toBe(true)
    expect(isValidHeaderList('Bad Header')).toBe(false)
  })
})
