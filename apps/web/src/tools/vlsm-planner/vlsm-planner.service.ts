import type { Cidr } from '@/utils/ip'
import { bitLength, cidrLast, cidrSize, cidrToString, formatCount, formatIP, parseCidr, rangeToCidrs } from '@/utils/ip'

export const MAX_SUBNETS = 256

export interface Requirement { name: string, hosts: number }

export interface Allocation {
  name: string
  hosts: number
  cidr: string
  prefix: number
  first: string
  last: string
  usable: string
  /** Share of the allocated block that is needed (0..1). */
  efficiency: number
}

export type VlsmError = 'network' | 'requirements' | 'too-many' | 'no-fit'

export interface VlsmResult {
  allocations: Allocation[]
  free: string[]
  used: bigint
  total: bigint
}

/** Parses lines like "LAN 50", "LAN: 50", "50" (unnamed rows get "Subnet n"). */
export function parseRequirements(input: string): Requirement[] | undefined {
  const out: Requirement[] = []
  for (const raw of input.split(/[\n,;]+/)) {
    const line = raw.trim()
    if (!line)
      continue
    const count = /\d{1,12}$/.exec(line)
    const rest = count ? line.slice(0, count.index) : ''
    // The name must be separated from the count ("LAN50" is not "LAN 50").
    if (!count || Number(count[0]) < 1 || (rest && !/[\s:=]$/.test(rest)))
      return undefined
    const name = rest.replace(/[\s:=]+$/, '').trim()
    out.push({ name: name || `Subnet ${out.length + 1}`, hosts: Number(count[0]) })
  }
  return out.length ? out : undefined
}

/**
 * Prefix length needed for `hosts` usable addresses. IPv4 reserves the network and broadcast
 * address (except /31 for 2 hosts, RFC 3021, and /32 for 1); IPv6 has no broadcast.
 */
export function prefixFor(version: 4 | 6, hosts: number, pointToPoint: boolean): number {
  const bits = bitLength(version)
  if (version === 4 && pointToPoint && hosts <= 2)
    return hosts === 1 ? 32 : 31
  const needed = BigInt(version === 4 ? hosts + 2 : hosts)
  let size = 1n
  let p = bits
  while (size < needed) {
    size <<= 1n
    p--
  }
  return p
}

function usableRange(c: Cidr): [bigint, bigint] {
  const last = cidrLast(c)
  if (c.version === 4 && c.prefix < 31)
    return [c.network + 1n, last - 1n]
  return [c.network, last]
}

/**
 * Largest-first allocation: blocks sorted by size and placed back to back from the start of the
 * network stay aligned to their own size, which yields the classic VLSM plan.
 */
export function plan(network: string, input: string, pointToPoint = true): VlsmResult | VlsmError {
  const net = parseCidr(network, { allowBare: false })
  if (!net)
    return 'network'
  const reqs = parseRequirements(input)
  if (!reqs)
    return 'requirements'
  if (reqs.length > MAX_SUBNETS)
    return 'too-many'
  const sorted = reqs.map((r, i) => ({ ...r, i, prefix: prefixFor(net.version, r.hosts, pointToPoint) }))
    .sort((a, b) => a.prefix - b.prefix || a.i - b.i)
  const end = cidrLast(net)
  let cur = net.network
  const allocations: Allocation[] = []
  for (const r of sorted) {
    if (r.prefix < net.prefix)
      return 'no-fit'
    const block: Cidr = { version: net.version, address: cur, network: cur, prefix: r.prefix }
    const last = cidrLast(block)
    if (last > end)
      return 'no-fit'
    const [first, lastUsable] = usableRange(block)
    const usable = lastUsable - first + 1n
    allocations.push({
      name: r.name,
      hosts: r.hosts,
      cidr: cidrToString(block),
      prefix: r.prefix,
      first: formatIP(net.version, first),
      last: formatIP(net.version, lastUsable),
      usable: formatCount(usable),
      efficiency: Number(BigInt(r.hosts) * 10000n / usable) / 10000,
    })
    cur = last + 1n
  }
  const used = cur - net.network
  return {
    allocations,
    free: cur <= end ? rangeToCidrs(net.version, cur, end).map(cidrToString) : [],
    used,
    total: cidrSize(net),
  }
}
