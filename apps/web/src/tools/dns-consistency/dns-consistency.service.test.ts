import type { DnsConsistency } from './dns-consistency.service'
import { describe, expect, it } from 'vitest'
import { checks } from './dns-consistency.service'

const server = (error = '') => ({ name: 'ns.', address: '192.0.2.1', serial: 1, mname: '', rname: '', authoritative: true, nsSet: [], rttMs: 1, error })

describe('checks', () => {
  it('flags unreachable servers', () => {
    const r: DnsConsistency = { zone: 'x', nameservers: ['ns.'], servers: [server(), server('no answer')], serialsConsistent: true, nsSetsConsistent: true, allAuthoritative: true }
    expect(checks(r).map(c => [c.id, c.ok])).toEqual([['serials', true], ['nsSets', true], ['authoritative', true], ['reachable', false]])
  })
})
