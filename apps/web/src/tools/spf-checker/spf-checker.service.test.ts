import { describe, expect, it } from 'vitest'
import { flatten } from './spf-checker.service'

describe('flatten', () => {
  it('indents includes', () => {
    const tree = flatten({ domain: 'a', record: '', error: '', terms: [
      { raw: 'include:b', qualifier: '+', kind: 'include', value: 'b', lookup: true, include: { domain: 'b', record: '', error: '', terms: [{ raw: 'ip4:192.0.2.0/24', qualifier: '+', kind: 'ip4', value: '', lookup: false }] } },
      { raw: '-all', qualifier: '-', kind: 'all', value: '', lookup: false },
    ] })
    expect(tree.map(l => `${l.depth}:${l.text}`)).toEqual(['0:a', '1:include:b', '2:b', '3:ip4:192.0.2.0/24', '1:-all'])
  })
})
