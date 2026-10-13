import { expandIPv6, formatIPv6, parseIPv6 } from '@/utils/ip'
import { classify } from '@/utils/special-ranges'

export interface Ipv6Forms {
  compressed: string
  expanded: string
  expandedNoZeros: string
  reverse: string
  uri: string
  hex: string
  decimal: string
  type: string
}

/** Full 32-nibble ip6.arpa pointer name. */
export function ip6Arpa(value: bigint): string {
  return `${[...value.toString(16).padStart(32, '0')].reverse().join('.')}.ip6.arpa`
}

export function ipv6Forms(input: string): Ipv6Forms | undefined {
  const value = parseIPv6(input)
  if (value === undefined)
    return undefined
  const compressed = formatIPv6(value)
  const special = classify({ version: 6, value }).primary
  return {
    compressed,
    expanded: expandIPv6(value),
    expandedNoZeros: expandIPv6(value).split(':').map(g => g.replace(/^0+(?=.)/, '')).join(':'),
    reverse: ip6Arpa(value),
    uri: `[${compressed}]`,
    hex: `0x${value.toString(16).padStart(32, '0')}`,
    decimal: value.toString(),
    type: special ? `${special.name} (${special.rfc})` : 'Global unicast',
  }
}
