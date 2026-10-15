import type { components } from '@/api/schema'

export type Dhcpv6Result = components['schemas']['Dhcpv6Result']

export type Verdict = 'none' | 'single' | 'multiple'
export const verdict = (r: Dhcpv6Result): Verdict => r.servers === 0 ? 'none' : r.servers === 1 ? 'single' : 'multiple'
