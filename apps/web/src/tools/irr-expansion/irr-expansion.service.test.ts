import type { IrrResult } from './irr-expansion.service'
import { describe, expect, it } from 'vitest'
import { prefixList } from '../router-snippets/router-snippets.service'
import { isSetName, listName, plain, toEntries } from './irr-expansion.service'

const r: IrrResult = { set: 'AS13335', mode: 'prefixes', family: 4, aggregate: true, sources: [], asns: [], prefixes: [{ prefix: '1.0.0.0/24', exact: true }, { prefix: '5.10.214.0/23', exact: false, greaterEqual: 24, lessEqual: 24 }] }

describe('irr-expansion', () => {
  it('validates set names', () => {
    expect(isSetName('AS-EXAMPLE')).toBe(true)
    expect(isSetName('as65000:as-customers')).toBe(true)
    expect(isSetName('AS65000')).toBe(true)
    expect(isSetName('-h evil')).toBe(false)
    expect(isSetName('FOO')).toBe(false)
  })
  it('feeds the snippet generator', () => {
    expect(prefixList(toEntries(r), 'ios', listName('AS65000:AS-X'))).toBe('ip prefix-list AS65000-AS-X seq 5 permit 1.0.0.0/24\nip prefix-list AS65000-AS-X seq 10 permit 5.10.214.0/23 ge 24 le 24')
    expect(plain(r)).toBe('1.0.0.0/24\n5.10.214.0/23 24-24')
    expect(plain({ ...r, mode: 'asns', asns: [26, 31] })).toBe('AS26\nAS31')
  })
})
