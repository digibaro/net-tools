import { describe, expect, it } from 'vitest'
import { parseIPv4 } from '@/utils/ip'
import { calculateIpv4Subnet, maskToPrefix, parseIpv4Input } from './ipv4-subnet-calculator.service'

function calc(s: string) {
  const p = parseIpv4Input(s)!
  return calculateIpv4Subnet(p.address, p.prefix)
}

describe('parseIpv4Input', () => {
  it.each([
    ['192.168.1.10/24', 24],
    ['192.168.1.10 / 24', 24],
    ['192.168.1.10 255.255.255.0', 24],
    ['192.168.1.10/255.255.240.0', 20],
    ['10.0.0.1', 32],
    ['10.0.0.1/0', 0],
  ])('%s → /%i', (input, prefix) => {
    expect(parseIpv4Input(input)?.prefix).toBe(prefix)
  })
  it.each(['', '192.168.1.10/33', '192.168.1.10 255.0.255.0', '300.1.1.1/24', '1.2.3.4/24/1', '1.2.3.4 5 6'])('rejects %j', (s) => {
    expect(parseIpv4Input(s)).toBeUndefined()
  })
})

describe('maskToPrefix', () => {
  it.each([['0.0.0.0', 0], ['128.0.0.0', 1], ['255.255.255.252', 30], ['255.255.255.255', 32]])('%s → %i', (m, p) => {
    expect(maskToPrefix(m)).toBe(p)
  })
  it('rejects non-contiguous masks', () => {
    expect(maskToPrefix('255.0.255.0')).toBeUndefined()
  })
})

describe('calculateIpv4Subnet', () => {
  it('computes a /24', () => {
    expect(calc('192.168.1.10/24')).toMatchObject({
      network: '192.168.1.0',
      broadcast: '192.168.1.255',
      netmask: '255.255.255.0',
      wildcard: '0.0.0.255',
      firstHost: '192.168.1.1',
      lastHost: '192.168.1.254',
      totalAddresses: '256',
      usableHosts: '254',
      addressClass: 'C',
      type: 'Private-Use (RFC 1918)',
      binaryNetmask: '11111111.11111111.11111111.00000000',
      hexAddress: '0xC0A8010A',
      integerAddress: String(parseIPv4('192.168.1.10')),
      previousNetwork: '192.168.0.0/24',
      nextNetwork: '192.168.2.0/24',
    })
  })

  it('handles /31 and /32 (RFC 3021)', () => {
    expect(calc('10.0.0.1/31')).toMatchObject({ network: '10.0.0.0', broadcast: null, firstHost: '10.0.0.0', lastHost: '10.0.0.1', usableHosts: '2' })
    expect(calc('8.8.8.8')).toMatchObject({ network: '8.8.8.8', usableHosts: '1', type: 'Public', addressClass: 'A' })
  })

  it('handles the edges of the address space', () => {
    expect(calc('0.0.0.0/0')).toMatchObject({ totalAddresses: '4,294,967,296', previousNetwork: null, nextNetwork: null })
    expect(calc('255.255.255.0/24').nextNetwork).toBeNull()
    expect(calc('0.0.0.5/24').previousNetwork).toBeNull()
  })

  it('classifies', () => {
    expect(calc('224.0.0.1/4').addressClass).toBe('D')
    expect(calc('100.64.0.1/10').type).toBe('Shared Address Space (CGNAT) (RFC 6598)')
  })
})
