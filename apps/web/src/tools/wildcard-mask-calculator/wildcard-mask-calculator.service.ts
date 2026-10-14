import { formatCount, formatIPv4, parseIPv4, prefixMask } from '@/utils/ip'

const FULL = 0xFFFFFFFFn

export type MaskKind = 'prefix' | 'netmask' | 'wildcard'

export interface MaskResult {
  kind: MaskKind
  prefix: number | null
  netmask: string
  wildcard: string
  binaryWildcard: string
  /** Contiguous masks map to a prefix; non-contiguous wildcards are valid in ACLs only. */
  contiguous: boolean
  matchingAddresses: string
  aclExample: string | null
  ospfExample: string | null
}

function contiguousPrefix(mask: bigint): number | undefined {
  for (let p = 0; p <= 32; p++) {
    if (prefixMask(4, p) === mask)
      return p
  }
  return undefined
}

const popcount = (v: bigint) => [...v.toString(2)].filter(c => c === '1').length

/**
 * "/24" or "24" → prefix; dotted value: a contiguous netmask (leading ones) is a netmask,
 * anything else is treated as a wildcard (inverse) mask.
 */
export function analyzeMask(input: string, network = ''): MaskResult | undefined {
  const s = input.trim()
  let kind: MaskKind
  let wildcard: bigint
  const p = /^\/?(\d{1,2})$/.exec(s)
  if (p) {
    if (Number(p[1]) > 32)
      return undefined
    kind = 'prefix'
    wildcard = FULL ^ prefixMask(4, Number(p[1]))
  }
  else {
    const v = parseIPv4(s)
    if (v === undefined)
      return undefined
    const asNetmask = contiguousPrefix(v)
    if (asNetmask !== undefined && v !== 0n) {
      kind = 'netmask'
      wildcard = FULL ^ v
    }
    else {
      kind = 'wildcard'
      wildcard = v
    }
  }
  const netmask = FULL ^ wildcard
  const prefix = contiguousPrefix(netmask) ?? null
  const base = parseIPv4(network)
  const net = base === undefined ? undefined : formatIPv4(base & netmask)
  return {
    kind,
    prefix,
    netmask: formatIPv4(netmask),
    wildcard: formatIPv4(wildcard),
    binaryWildcard: wildcard.toString(2).padStart(32, '0').replace(/(.{8})(?!$)/g, '$1.'),
    contiguous: prefix !== null,
    matchingAddresses: formatCount(1n << BigInt(popcount(wildcard))),
    aclExample: net ? `access-list 10 permit ${net} ${formatIPv4(wildcard)}` : null,
    ospfExample: net && prefix !== null ? `network ${net} ${formatIPv4(wildcard)} area 0` : null,
  }
}
