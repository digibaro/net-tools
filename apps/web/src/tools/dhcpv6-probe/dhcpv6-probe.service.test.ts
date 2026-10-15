import type { Dhcpv6Result } from './dhcpv6-probe.service'
import { describe, expect, it } from 'vitest'
import { verdict } from './dhcpv6-probe.service'

describe('dhcpv6-probe', () => {
  it('rates the number of servers', () => {
    expect(verdict({ servers: 0 } as Dhcpv6Result)).toBe('none')
    expect(verdict({ servers: 1 } as Dhcpv6Result)).toBe('single')
    expect(verdict({ servers: 3 } as Dhcpv6Result)).toBe('multiple')
  })
})
