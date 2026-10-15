import type { Cidr, IPVersion } from '@/utils/ip'
import { bitLength, cidrLast, cidrSize, cidrToString, formatCount, formatIP, parseCidr, prefixMask, rangeToCidrs } from '@/utils/ip'
import { prefixFor } from '../vlsm-planner/vlsm-planner.service'

export const MAX_ROWS = 512

export interface PlanLine { line: number, name: string, vlan: string, size: string, fixed: string, note: string }
export type RowError = 'size' | 'fixed' | 'version' | 'outside' | 'overlap' | 'no-space'

export interface PlanRow {
  line: number
  name: string
  vlan: string
  note: string
  fixed: boolean
  cidr?: string
  gateway?: string
  range?: string
  mask?: string
  usable?: string
  error?: RowError
}

export interface PlanResult {
  version: IPVersion
  block: string
  rows: PlanRow[]
  free: string[]
  used: bigint
  total: bigint
}

/** "Name, VLAN, size, [fixed network], [note]" per line; fields split on comma, tab, semicolon or |. */
export function parsePlan(input: string): PlanLine[] {
  const out: PlanLine[] = []
  input.split('\n').forEach((raw, i) => {
    const line = raw.trim()
    if (!line || line.startsWith('#'))
      return
    const [name = '', vlan = '', size = '', fixed = '', ...note] = line.split(/\s*[,\t;|]\s*/)
    out.push({ line: i + 1, name, vlan, size, fixed, note: note.join(', ') })
  })
  return out
}

/** "/26" or "26" = prefix length; "h50" or "50 hosts" = hosts needed. A bare number above the address width counts as hosts. */
export function sizeToPrefix(version: IPVersion, size: string): number | undefined {
  const s = size.trim().toLowerCase()
  const bits = bitLength(version)
  const prefix = /^\/(\d{1,3})$/.exec(s)
  if (prefix)
    return Number(prefix[1]) <= bits ? Number(prefix[1]) : undefined
  const hosts = /^(?:h\s*)?(\d{1,12})(?:\s*hosts?)?$/.exec(s)
  if (!hosts)
    return undefined
  const n = Number(hosts[1])
  if (/^\d+$/.test(s) && n <= bits && !s.startsWith('h'))
    return n
  return n >= 1 ? prefixFor(version, n, true) : undefined
}

interface Span { start: bigint, end: bigint }

function describe(version: IPVersion, c: Cidr): Pick<PlanRow, 'cidr' | 'gateway' | 'range' | 'mask' | 'usable'> {
  const last = cidrLast(c)
  const [first, lastUsable] = version === 4 && c.prefix < 31 ? [c.network + 1n, last - 1n] : [c.network, last]
  return {
    cidr: cidrToString(c),
    gateway: formatIP(version, version === 6 && c.prefix < 127 ? c.network + 1n : first),
    range: `${formatIP(version, first)} – ${formatIP(version, lastUsable)}`,
    mask: version === 4 ? formatIP(4, prefixMask(4, c.prefix)) : `/${c.prefix}`,
    usable: formatCount(lastUsable - first + 1n),
  }
}

export function buildPlan(blockInput: string, planInput: string): PlanResult | 'block' | 'too-many' {
  const net = parseCidr(blockInput.trim(), { allowBare: false })
  if (!net)
    return 'block'
  const lines = parsePlan(planInput)
  if (lines.length > MAX_ROWS)
    return 'too-many'
  const end = cidrLast(net)
  const taken: Span[] = []
  const overlaps = (s: Span) => taken.some(t => s.start <= t.end && t.start <= s.end)
  const take = (s: Span) => {
    taken.push(s)
    taken.sort((a, b) => (a.start < b.start ? -1 : 1))
  }
  const rows: PlanRow[] = lines.map(l => ({ line: l.line, name: l.name, vlan: l.vlan, note: l.note, fixed: !!l.fixed }))
  // Fixed networks first, so automatic ones flow around them.
  lines.forEach((l, i) => {
    if (!l.fixed)
      return
    const row = rows[i]!
    const c = parseCidr(l.fixed, { allowBare: false })
    if (!c)
      return void (row.error = 'fixed')
    if (c.version !== net.version)
      return void (row.error = 'version')
    const span = { start: c.network, end: cidrLast(c) }
    if (c.prefix < net.prefix || span.start < net.network || span.end > end)
      return void (row.error = 'outside')
    if (overlaps(span))
      return void (row.error = 'overlap')
    take(span)
    Object.assign(row, describe(net.version, c))
  })
  lines.forEach((l, i) => {
    if (l.fixed)
      return
    const row = rows[i]!
    const p = sizeToPrefix(net.version, l.size)
    if (p === undefined)
      return void (row.error = 'size')
    if (p < net.prefix)
      return void (row.error = 'no-space')
    const size = 1n << BigInt(bitLength(net.version) - p)
    let cur = net.network
    for (const t of [...taken, { start: end + 1n, end: end + 1n }]) {
      const start = (cur + size - 1n) / size * size
      if (start + size - 1n < t.start && start + size - 1n <= end) {
        take({ start, end: start + size - 1n })
        Object.assign(row, describe(net.version, { version: net.version, address: start, network: start, prefix: p }))
        return
      }
      if (t.end + 1n > cur)
        cur = t.end + 1n
    }
    row.error = 'no-space'
  })
  const free: string[] = []
  let cur = net.network
  let used = 0n
  for (const t of taken) {
    if (t.start > cur)
      free.push(...rangeToCidrs(net.version, cur, t.start - 1n).map(cidrToString))
    used += t.end - t.start + 1n
    cur = t.end + 1n
  }
  if (cur <= end)
    free.push(...rangeToCidrs(net.version, cur, end).map(cidrToString))
  return { version: net.version, block: cidrToString(net), rows, free, used, total: cidrSize(net) }
}
