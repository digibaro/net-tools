export interface ParsedUrl {
  href: string
  origin: string
  protocol: string
  username: string
  hasPassword: boolean
  hostname: string
  port: string
  effectivePort: string
  pathname: string
  search: string
  hash: string
  params: { key: string, value: string }[]
}

const DEFAULT_PORTS: Record<string, string> = { 'http:': '80', 'https:': '443', 'ws:': '80', 'wss:': '443', 'ftp:': '21' }

export function parseUrl(input: string): ParsedUrl | undefined {
  const s = input.trim()
  if (!s)
    return undefined
  let u: URL
  try {
    u = new URL(s)
  }
  catch {
    return undefined
  }
  return {
    href: u.href,
    origin: u.origin,
    protocol: u.protocol,
    username: decodeURIComponent(u.username),
    hasPassword: u.password !== '',
    hostname: u.hostname,
    port: u.port,
    effectivePort: u.port || DEFAULT_PORTS[u.protocol] || '',
    pathname: u.pathname,
    search: u.search,
    hash: u.hash,
    params: [...u.searchParams].map(([key, value]) => ({ key, value })),
  }
}

export type Codec = 'component' | 'uri'

export function encode(text: string, codec: Codec): string {
  return codec === 'component' ? encodeURIComponent(text) : encodeURI(text)
}

/** Decodes percent-escapes; `+` is treated as space (form encoding). Undefined on malformed input. */
export function decode(text: string, codec: Codec): string | undefined {
  try {
    const t = text.replaceAll('+', ' ')
    return codec === 'component' ? decodeURIComponent(t) : decodeURI(t)
  }
  catch {
    return undefined
  }
}
