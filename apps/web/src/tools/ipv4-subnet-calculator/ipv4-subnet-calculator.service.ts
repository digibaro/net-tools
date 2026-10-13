import type { IPAddress } from '@/utils/ip'
import { bitLength, formatCount, formatIPv4, hostMask, parseIPv4, prefixMask } from '@/utils/ip'
import { classify } from '@/utils/special-ranges'

export interface Ipv4Subnet {
  address: string
  prefix: number
  network: string
  broadcast: string | null
  netmask: string
  wildcard: string
  firstHost: string
  lastHost: string
  totalAddresses: string
  usableHosts: string
  addressClass: 'A' | 'B' | 'C' | 'D' | 'E'
  type: string
  binaryAddress: string
  binaryNetmask: string
  hexAddress: string
  integerAddress: string
  previousNetwork: string | null
  nextNetwork: string | null
}

/** Netmask (contiguous) → prefix length, or undefined. */
export function maskToPrefix(mask: string): number | undefined {
  const v = parseIPv4(mask)
  if (v === undefined)
    return undefined
  for (let prefix = 0; prefix <= 32; prefix++) {
    if (prefixMask(4, prefix) === v)
      return prefix
  }
  return undefined
}

/** Accepts "a.b.c.d/len", "a.b.c.d/mask", "a.b.c.d mask" or a bare address (/32). */
export function parseIpv4Input(input: string): { address: bigint, prefix: number } | undefined {
  const parts = input.trim().split(/\s*\/\s*|\s+/)
  if (parts.length > 2 || !parts[0])
    return undefined
  const address = parseIPv4(parts[0])
  if (address === undefined)
    return undefined
  if (parts.length === 1)
    return { address, prefix: 32 }
  const p = parts[1]!
  if (/^\d{1,2}$/.test(p) && Number(p) <= 32)
    return { address, prefix: Number(p) }
  const prefix = maskToPrefix(p)
  return prefix === undefined ? undefined : { address, prefix }
}

const bin = (v: bigint) => v.toString(2).padStart(32, '0').replace(/(.{8})(?!$)/g, '$1.')

export function addressClass(v: bigint): Ipv4Subnet['addressClass'] {
  const first = Number(v >> 24n)
  return first < 128 ? 'A' : first < 192 ? 'B' : first < 224 ? 'C' : first < 240 ? 'D' : 'E'
}

export function calculateIpv4Subnet(address: bigint, prefix: number): Ipv4Subnet {
  const mask = prefixMask(4, prefix)
  const network = address & mask
  const broadcast = network | hostMask(4, prefix)
  const size = 1n << BigInt(bitLength(4) - prefix)
  // RFC 3021: /31 has two usable addresses, /32 one; otherwise exclude network + broadcast.
  const usable = prefix >= 31 ? size : size - 2n
  const first = prefix >= 31 ? network : network + 1n
  const last = prefix >= 31 ? broadcast : broadcast - 1n
  const addr: IPAddress = { version: 4, value: address }
  const special = classify(addr).primary

  return {
    address: formatIPv4(address),
    prefix,
    network: formatIPv4(network),
    broadcast: prefix >= 31 ? null : formatIPv4(broadcast),
    netmask: formatIPv4(mask),
    wildcard: formatIPv4(hostMask(4, prefix)),
    firstHost: formatIPv4(first),
    lastHost: formatIPv4(last),
    totalAddresses: formatCount(size),
    usableHosts: formatCount(usable),
    addressClass: addressClass(address),
    type: special ? `${special.name} (${special.rfc})` : 'Public',
    binaryAddress: bin(address),
    binaryNetmask: bin(mask),
    hexAddress: `0x${address.toString(16).toUpperCase().padStart(8, '0')}`,
    integerAddress: address.toString(),
    previousNetwork: network === 0n ? null : `${formatIPv4((network - size) & mask)}/${prefix}`,
    nextNetwork: broadcast === 0xFFFFFFFFn ? null : `${formatIPv4(broadcast + 1n)}/${prefix}`,
  }
}
