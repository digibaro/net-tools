import type { Cidr } from '@/utils/ip'
import { bitLength, cidrToString, parseCidr, splitList } from '@/utils/ip'

export const MAX_ZONES = 256

export interface ReverseRow {
  input: string
  kind: 'ptr' | 'zone' | 'classless'
  names: string[]
  note?: string
}

function v4Labels(value: bigint, octets: number): string {
  return [24n, 16n, 8n, 0n].slice(0, octets).map(s => String((value >> s) & 0xFFn)).reverse().join('.')
}

function v6Labels(value: bigint, nibbles: number): string {
  return [...value.toString(16).padStart(32, '0').slice(0, nibbles)].reverse().join('.')
}

const zone = (labels: string, suffix: string) => (labels ? `${labels}.${suffix}` : suffix)

/** Zones covering the prefix at the next label boundary (octet for v4, nibble for v6). */
function coveringZones(c: Cidr): { names: string[], total: number } {
  const unit = c.version === 4 ? 8 : 4
  const boundary = Math.ceil(c.prefix / unit) * unit
  const total = 2 ** (boundary - c.prefix)
  const step = 1n << BigInt(bitLength(c.version) - boundary)
  const names: string[] = []
  for (let i = 0; i < Math.min(total, MAX_ZONES); i++) {
    const net = c.network + BigInt(i) * step
    names.push(c.version === 4 ? zone(v4Labels(net, boundary / 8), 'in-addr.arpa') : zone(v6Labels(net, boundary / 4), 'ip6.arpa'))
  }
  return { names, total }
}

export function reverseName(input: string): ReverseRow | undefined {
  const c = parseCidr(input)
  if (!c)
    return undefined
  const isHost = !input.includes('/') || c.prefix === bitLength(c.version)
  if (isHost) {
    const name = c.version === 4 ? `${v4Labels(c.address, 4)}.in-addr.arpa` : `${v6Labels(c.address, 32)}.ip6.arpa`
    return { input: input.trim(), kind: 'ptr', names: [name] }
  }
  const label = cidrToString(c)
  if (c.version === 4 && c.prefix > 24) {
    // RFC 2317 classless delegation: CNAMEs into a sub-zone named "<first>/<len>" (convention).
    const parent = zone(v4Labels(c.network, 3), 'in-addr.arpa')
    return { input: label, kind: 'classless', names: [`${c.network & 0xFFn}/${c.prefix}.${parent}`], note: parent }
  }
  const { names, total } = coveringZones(c)
  return { input: label, kind: 'zone', names, note: total > names.length ? String(total) : undefined }
}

export function reverseNames(input: string): { rows: ReverseRow[], invalid: string[] } {
  const rows: ReverseRow[] = []
  const invalid: string[] = []
  for (const s of splitList(input).slice(0, 1000)) {
    const r = reverseName(s)
    if (r)
      rows.push(r)
    else
      invalid.push(s)
  }
  return { rows, invalid }
}
