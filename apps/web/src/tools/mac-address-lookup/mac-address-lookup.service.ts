import { formatMac, isLocallyAdministered, isMulticast, ouiOf, parseMac } from '@/utils/mac'

export interface OuiEntry {
  vendor: string
  country: string
}

export type OuiTable = Map<string, OuiEntry>

/** Parses the bundled "OUI<TAB>vendor<TAB>country" table (scripts/gen-data.mjs). */
export function parseOuiTable(tsv: string): OuiTable {
  const table: OuiTable = new Map()
  for (const line of tsv.split('\n')) {
    const [oui, vendor = '', country = ''] = line.split('\t')
    if (oui)
      table.set(oui, { vendor, country })
  }
  return table
}

let cached: Promise<OuiTable> | undefined
export function loadOuiTable(): Promise<OuiTable> {
  cached ??= import('./oui.tsv?raw').then(m => parseOuiTable(m.default))
  return cached
}

/** Accepts a full MAC or just its first three octets (OUI). */
export function parseMacOrOui(input: string): { mac?: bigint, oui: string } | undefined {
  const mac = parseMac(input)
  if (mac !== undefined)
    return { mac, oui: ouiOf(mac) }
  const s = input.trim().toLowerCase().replace(/[:.-]/g, '')
  return /^[0-9a-f]{6}$/.test(s) ? { oui: s.toUpperCase() } : undefined
}

export interface LookupRow {
  input: string
  mac: string
  oui: string
  vendor: string
  country: string
  administration: 'universal' | 'local'
  cast: 'unicast' | 'multicast' | 'broadcast'
}

export function lookup(input: string, table: OuiTable): LookupRow | undefined {
  const p = parseMacOrOui(input)
  if (!p)
    return undefined
  const first = BigInt(`0x${p.oui}`) << 24n
  const bits = p.mac ?? first
  const entry = table.get(p.oui)
  return {
    input: input.trim(),
    mac: p.mac === undefined ? '' : formatMac(p.mac),
    oui: p.oui.match(/../g)!.join(':'),
    vendor: entry?.vendor ?? '',
    country: entry?.country ?? '',
    administration: isLocallyAdministered(bits) ? 'local' : 'universal',
    cast: p.mac === 0xFFFFFFFFFFFFn ? 'broadcast' : isMulticast(bits) ? 'multicast' : 'unicast',
  }
}
