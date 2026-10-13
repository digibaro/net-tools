import type { components } from '@/api/schema'

export type EmailSpf = components['schemas']['EmailSpf']
type SpfRecord = components['schemas']['SpfRecord']

export interface TreeLine { depth: number, text: string, lookup: boolean, error: string }

/** Flattens the include tree into indented lines. */
export function flatten(rec: SpfRecord, depth = 0): TreeLine[] {
  const out: TreeLine[] = [{ depth, text: rec.domain, lookup: false, error: rec.error }]
  for (const t of rec.terms) {
    out.push({ depth: depth + 1, text: t.raw, lookup: t.lookup, error: '' })
    if (t.include)
      out.push(...flatten(t.include, depth + 2))
  }
  return out
}
