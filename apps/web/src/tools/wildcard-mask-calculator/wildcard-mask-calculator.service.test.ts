import { describe, expect, it } from 'vitest'
import { analyzeMask } from './wildcard-mask-calculator.service'

describe('analyzeMask', () => {
  it.each([
    ['/24', 'prefix'],
    ['24', 'prefix'],
    ['255.255.255.0', 'netmask'],
    ['0.0.0.255', 'wildcard'],
  ])('%s is read as %s and maps to /24', (input, kind) => {
    expect(analyzeMask(input)).toMatchObject({ kind, prefix: 24, netmask: '255.255.255.0', wildcard: '0.0.0.255', contiguous: true, matchingAddresses: '256' })
  })

  it('handles the extremes', () => {
    expect(analyzeMask('0.0.0.0')).toMatchObject({ kind: 'wildcard', prefix: 32, matchingAddresses: '1' })
    expect(analyzeMask('/0')).toMatchObject({ wildcard: '255.255.255.255', prefix: 0 })
    expect(analyzeMask('255.255.255.255')).toMatchObject({ kind: 'netmask', prefix: 32 })
  })

  it('accepts non-contiguous ACL wildcards', () => {
    expect(analyzeMask('0.0.255.0')).toMatchObject({ kind: 'wildcard', prefix: null, contiguous: false, matchingAddresses: '256', netmask: '255.255.0.255' })
  })

  it('builds ACL and OSPF examples from a network', () => {
    const r = analyzeMask('/22', '10.1.7.9')!
    expect(r.aclExample).toBe('access-list 10 permit 10.1.4.0 0.0.3.255')
    expect(r.ospfExample).toBe('network 10.1.4.0 0.0.3.255 area 0')
    expect(analyzeMask('0.0.255.0', '10.1.7.9')!.ospfExample).toBeNull()
  })

  it.each(['/33', '256.0.0.0', 'abc', ''])('rejects %j', (s) => {
    expect(analyzeMask(s)).toBeUndefined()
  })
})
