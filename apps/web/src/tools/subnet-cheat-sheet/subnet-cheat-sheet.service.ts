import { formatCount, formatIPv4, hostMask, prefixMask } from '@/utils/ip'

export interface V4Row {
  prefix: string
  netmask: string
  wildcard: string
  addresses: string
  usable: string
  size: string
}

function sizeVs24(prefix: number): string {
  if (prefix <= 24)
    return `${formatCount(1n << BigInt(24 - prefix))} × /24`
  return `1/${2 ** (prefix - 24)} of a /24`
}

export function ipv4CheatSheet(): V4Row[] {
  return Array.from({ length: 33 }, (_, p) => {
    const size = 1n << BigInt(32 - p)
    return {
      prefix: `/${p}`,
      netmask: formatIPv4(prefixMask(4, p)),
      wildcard: formatIPv4(hostMask(4, p)),
      addresses: formatCount(size),
      usable: formatCount(p >= 31 ? size : size - 2n),
      size: sizeVs24(p),
    }
  })
}

export interface V6Row {
  prefix: string
  subnets64: string
  addresses: string
  use: string
}

/** Common IPv6 prefix lengths; `use` is an i18n key suffix. */
export const V6_PREFIXES = [128, 127, 126, 124, 120, 112, 96, 64, 60, 56, 52, 48, 44, 40, 36, 32, 29, 24, 20, 16, 12, 8] as const

export function ipv6CheatSheet(): Omit<V6Row, 'use'>[] {
  return V6_PREFIXES.map(p => ({
    prefix: `/${p}`,
    subnets64: p <= 64 ? formatCount(1n << BigInt(64 - p)) : '—',
    addresses: formatCount(1n << BigInt(128 - p)),
  }))
}
