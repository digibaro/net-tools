import type { DhcpResult } from './dhcp-probe.service'
import { describe, expect, it } from 'vitest'
import { formatLease, isInterfaceName, verdict } from './dhcp-probe.service'

describe('dhcp-probe', () => {
  it('validates interface names', () => {
    expect(isInterfaceName('')).toBe(true)
    expect(isInterfaceName('enp3s0')).toBe(true)
    expect(isInterfaceName('br-lan.10')).toBe(true)
    expect(isInterfaceName('../etc')).toBe(false)
    expect(isInterfaceName('a b')).toBe(false)
  })
  it('formats leases and verdicts', () => {
    expect(formatLease(86400)).toBe('1 d')
    expect(formatLease(7200)).toBe('2 h')
    expect(formatLease(90)).toBe('90 s')
    expect(verdict({ servers: 2 } as DhcpResult)).toBe('multiple')
  })
})
