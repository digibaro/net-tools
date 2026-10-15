import type { Detection } from './cdn-waf-detector.service'
import { describe, expect, it } from 'vitest'
import { sortDetections } from './cdn-waf-detector.service'

describe('cdn-waf-detector', () => {
  it('lists CDNs before WAFs and hosting', () => {
    const d = [{ name: 'Vercel', kind: 'hosting', evidence: ['a'] }, { name: 'Imperva', kind: 'waf', evidence: ['a'] }, { name: 'Akamai', kind: 'cdn', evidence: ['a'] }] as Detection[]
    expect(sortDetections(d).map(x => x.name)).toEqual(['Akamai', 'Imperva', 'Vercel'])
  })
})
