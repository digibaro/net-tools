import { formatIPv6, parseCidr, parseIPv6 } from '@/utils/ip'
import { formatMac, macToEui64, parseMac } from '@/utils/mac'

const MASK64 = (1n << 64n) - 1n
const LINK_LOCAL = 0xFE80n << 112n

export interface Eui64Result {
  mac: string
  interfaceId: string
  linkLocal: string
  slaac: string | null
}

const iidText = (iid: bigint) => iid.toString(16).padStart(16, '0').match(/.{4}/g)!.join(':')

/** /64 prefix for SLAAC; must be exactly 64 bits long. */
export function parsePrefix64(input: string): bigint | undefined {
  if (!input.trim())
    return undefined
  const c = parseCidr(input.includes('/') ? input : `${input}/64`)
  return c?.version === 6 && c.prefix === 64 ? c.network : undefined
}

export function fromMac(macInput: string, prefixInput = ''): Eui64Result | undefined {
  const mac = parseMac(macInput)
  if (mac === undefined)
    return undefined
  const iid = macToEui64(mac)
  const prefix = parsePrefix64(prefixInput)
  return {
    mac: formatMac(mac),
    interfaceId: iidText(iid),
    linkLocal: formatIPv6(LINK_LOCAL | iid),
    slaac: prefix === undefined ? null : formatIPv6(prefix | iid),
  }
}

/** Recovers the MAC from an IPv6 address whose interface ID is a modified EUI-64. */
export function toMac(addressInput: string): string | undefined {
  const v = parseIPv6(addressInput)
  if (v === undefined)
    return undefined
  const iid = (v & MASK64) ^ (0x02n << 56n)
  if (((iid >> 24n) & 0xFFFFn) !== 0xFFFEn)
    return undefined
  return formatMac(((iid >> 40n) << 24n) | (iid & 0xFFFFFFn))
}
