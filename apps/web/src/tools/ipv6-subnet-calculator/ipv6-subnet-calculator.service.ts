import { cidrLast, cidrSize, expandIPv6, formatCount, formatIPv6, parseCidr } from '@/utils/ip'
import { classify } from '@/utils/special-ranges'

export interface Ipv6Subnet {
  address: string
  addressExpanded: string
  prefix: number
  network: string
  networkExpanded: string
  firstAddress: string
  lastAddress: string
  lastAddressExpanded: string
  totalAddresses: string
  subnets64: string
  subnets48: string
  reverseZone: string
  type: string
  previousNetwork: string | null
  nextNetwork: string | null
}

export function parseIpv6Input(input: string) {
  const c = parseCidr(input)
  return c?.version === 6 ? c : undefined
}

/** ip6.arpa zone for a prefix, truncated to the last whole nibble. */
export function reverseZone(network: bigint, prefix: number): string {
  const nibbles = Math.floor(prefix / 4)
  const hex = network.toString(16).padStart(32, '0').slice(0, nibbles)
  return `${[...hex].reverse().join('.')}${nibbles ? '.' : ''}ip6.arpa`
}

function countSubnets(prefix: number, target: number) {
  return prefix > target ? '0' : formatCount(1n << BigInt(target - prefix))
}

export function calculateIpv6Subnet(input: string): Ipv6Subnet | undefined {
  const c = parseIpv6Input(input)
  if (!c)
    return undefined
  const last = cidrLast(c)
  const size = cidrSize(c)
  const special = classify({ version: 6, value: c.address }).primary
  const max = (1n << 128n) - 1n
  return {
    address: formatIPv6(c.address),
    addressExpanded: expandIPv6(c.address),
    prefix: c.prefix,
    network: `${formatIPv6(c.network)}/${c.prefix}`,
    networkExpanded: expandIPv6(c.network),
    firstAddress: formatIPv6(c.network),
    lastAddress: formatIPv6(last),
    lastAddressExpanded: expandIPv6(last),
    totalAddresses: formatCount(size),
    subnets64: countSubnets(c.prefix, 64),
    subnets48: countSubnets(c.prefix, 48),
    reverseZone: reverseZone(c.network, c.prefix),
    type: special ? `${special.name} (${special.rfc})` : 'Global unicast',
    previousNetwork: c.network === 0n ? null : `${formatIPv6(c.network - size)}/${c.prefix}`,
    nextNetwork: last === max ? null : `${formatIPv6(last + 1n)}/${c.prefix}`,
  }
}
