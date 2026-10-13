import { COMMON_TYPES } from './dns-record-types.examples'
import rrtypes from './rrtypes.json'

export interface RrType {
  type: string
  value: number
  meaning: string
  rfc: string[]
}

export interface RrRow extends RrType {
  common: boolean
  summary: string
  example: string
}

export const RR_TYPES: RrRow[] = (rrtypes as RrType[]).map(r => ({
  ...r,
  common: r.type in COMMON_TYPES,
  summary: COMMON_TYPES[r.type]?.summary ?? r.meaning,
  example: COMMON_TYPES[r.type]?.example ?? '',
}))

export function filterTypes(query: string, commonOnly: boolean, list: RrRow[] = RR_TYPES): RrRow[] {
  const q = query.trim().toLowerCase()
  return list.filter(r =>
    (!commonOnly || r.common)
    && (!q || (/^\d+$/.test(q) ? r.value === Number(q) : r.type.toLowerCase().includes(q) || r.summary.toLowerCase().includes(q))))
}
