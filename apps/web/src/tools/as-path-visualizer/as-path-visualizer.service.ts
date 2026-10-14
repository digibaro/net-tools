/** One hop of a path: an ASN repeated `count` times (prepending), or an AS_SET. */
export interface Hop {
  asn: number
  count: number
  set?: number[]
}

export interface PathInfo {
  raw: string
  hops: Hop[]
  /** Length as BGP counts it (prepends included, an AS_SET counts as 1). */
  length: number
  origin?: number
  prepends: { asn: number, count: number, atOrigin: boolean }[]
  loops: number[]
  privateAsns: number[]
  reservedAsns: number[]
}

export type AsnClass = 'public' | 'private' | 'reserved' | 'documentation'

/** RFC 6996 private, RFC 5398 documentation, RFC 7300/4893/7607 reserved. */
export function classify(asn: number): AsnClass {
  if ((asn >= 64512 && asn <= 65534) || (asn >= 4200000000 && asn <= 4294967294))
    return 'private'
  if ((asn >= 64496 && asn <= 64511) || (asn >= 65536 && asn <= 65551))
    return 'documentation'
  if (asn === 0 || asn === 23456 || asn === 65535 || asn === 4294967295 || (asn >= 65552 && asn <= 131071))
    return 'reserved'
  return 'public'
}

const ASN_MAX = 4294967295

function num(tok: string): number | undefined {
  const m = /^(?:AS)?(\d{1,10})$/i.exec(tok)
  if (!m)
    return undefined
  const n = Number(m[1])
  return n <= ASN_MAX ? n : undefined
}

/**
 * Parses one AS path ("64500 64501 64501 13335", "AS64500 AS13335", "64500 {65001,65002}").
 * Origin codes (i, e, ?) and other words are ignored. Returns undefined without any ASN.
 */
export function parsePath(line: string): PathInfo | undefined {
  const hops: Hop[] = []
  const tokens = line.replace(/[{}]/g, m => ` ${m} `).replace(/,/g, ' ').trim().split(/\s+/)
  let inSet: number[] | undefined
  for (const tok of tokens) {
    if (tok === '{') {
      inSet = []
      continue
    }
    if (tok === '}') {
      if (inSet?.length)
        hops.push({ asn: inSet[0]!, count: 1, set: [...new Set(inSet)].sort((a, b) => a - b) })
      inSet = undefined
      continue
    }
    const n = num(tok)
    if (n === undefined)
      continue
    if (inSet) {
      inSet.push(n)
      continue
    }
    const last = hops.at(-1)
    if (last && !last.set && last.asn === n)
      last.count++
    else
      hops.push({ asn: n, count: 1 })
  }
  if (!hops.length)
    return undefined
  const seen = new Set<number>()
  const loops = new Set<number>()
  const all: number[] = []
  for (const h of hops) {
    for (const a of h.set ?? [h.asn]) {
      if (!h.set && seen.has(a))
        loops.add(a)
      all.push(a)
    }
    if (!h.set)
      seen.add(h.asn)
  }
  const uniq = [...new Set(all)]
  const last = hops.at(-1)!
  return {
    raw: line.trim(),
    hops,
    length: hops.reduce((n, h) => n + (h.set ? 1 : h.count), 0),
    origin: last.set ? undefined : last.asn,
    prepends: hops.filter(h => h.count > 1).map(h => ({ asn: h.asn, count: h.count, atOrigin: h === last })),
    loops: [...loops],
    privateAsns: uniq.filter(a => classify(a) === 'private'),
    reservedAsns: uniq.filter(a => ['reserved', 'documentation'].includes(classify(a))),
  }
}

export function parsePaths(input: string): PathInfo[] {
  return input.split('\n').map(parsePath).filter((p): p is PathInfo => !!p)
}

export interface GraphNode {
  asn: number
  label: string
  layer: number
  row: number
  isOrigin: boolean
  cls: AsnClass
  /** Highest prepend count this ASN used in any path. */
  prepend: number
}

export interface GraphEdge {
  from: number
  to: number
  paths: number
}

/**
 * Layered layout: layer 0 is the origin; each ASN sits at its greatest distance from the
 * origin over all paths, so shared upstreams line up. Rows follow first appearance.
 */
export function buildGraph(paths: PathInfo[]): { nodes: GraphNode[], edges: GraphEdge[], layers: number } {
  const layer = new Map<number, number>()
  const order: number[] = []
  const origins = new Set<number>()
  const prepend = new Map<number, number>()
  const edges = new Map<string, GraphEdge>()
  for (const p of paths) {
    const hops = p.hops.filter(h => !h.set)
    // Remove loops for drawing: keep the first occurrence closest to the origin.
    const chain: Hop[] = []
    for (const h of [...hops].reverse()) {
      if (!chain.some(c => c.asn === h.asn))
        chain.push(h)
    }
    chain.forEach((h, i) => {
      prepend.set(h.asn, Math.max(prepend.get(h.asn) ?? 1, h.count))
      if (!layer.has(h.asn))
        order.push(h.asn)
      layer.set(h.asn, Math.max(layer.get(h.asn) ?? 0, i))
    })
    if (chain[0])
      origins.add(chain[0].asn)
    for (let i = chain.length - 1; i > 0; i--) {
      const from = chain[i]!
      const to = chain[i - 1]!
      const key = `${from.asn}>${to.asn}`
      const e = edges.get(key) ?? { from: from.asn, to: to.asn, paths: 0 }
      e.paths++
      edges.set(key, e)
    }
  }
  const rows = new Map<number, number>()
  const nodes = order.map((asn) => {
    const l = layer.get(asn)!
    const row = rows.get(l) ?? 0
    rows.set(l, row + 1)
    return { asn, label: `AS${asn}`, layer: l, row, isOrigin: origins.has(asn), cls: classify(asn), prepend: prepend.get(asn) ?? 1 }
  })
  return { nodes, edges: [...edges.values()], layers: Math.max(0, ...nodes.map(n => n.layer)) + 1 }
}
