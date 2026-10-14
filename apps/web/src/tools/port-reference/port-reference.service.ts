/** [port or range, transport, service name, description, reference] from the IANA registry. */
export type PortEntry = [string, string, string, string, string]

export interface PortRow {
  port: string
  protocol: string
  service: string
  description: string
  reference: string
}

export const MAX_RESULTS = 300

let cached: Promise<PortEntry[]> | undefined
export function loadPorts(): Promise<PortEntry[]> {
  cached ??= import('./ports.json').then(m => m.default as PortEntry[])
  return cached
}

function portMatches(spec: string, port: number): boolean {
  const [a, b] = spec.split('-').map(Number)
  return b === undefined ? a === port : port >= a! && port <= b
}

/** Numeric query → exact port (incl. ranges); text → service name, then description. */
export function searchPorts(entries: PortEntry[], query: string, protocol: string): { rows: PortRow[], total: number } {
  const q = query.trim().toLowerCase()
  const proto = (p: string) => protocol === 'all' || p === protocol
  let hits: PortEntry[]
  if (!q) {
    hits = []
  }
  else if (/^\d{1,5}$/.test(q)) {
    const port = Number(q)
    hits = entries.filter(e => proto(e[1]) && portMatches(e[0], port))
  }
  else {
    const byName = entries.filter(e => proto(e[1]) && e[2].toLowerCase() === q)
    const partial = entries.filter(e => proto(e[1]) && e[2].toLowerCase() !== q && (e[2].toLowerCase().includes(q) || e[3].toLowerCase().includes(q)))
    hits = [...byName, ...partial]
  }
  return {
    total: hits.length,
    rows: hits.slice(0, MAX_RESULTS).map(([port, protocol, service, description, reference]) => ({ port, protocol, service, description, reference })),
  }
}
