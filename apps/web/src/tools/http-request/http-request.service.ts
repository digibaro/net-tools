import type { components } from '@/api/schema'

export type HttpHops = components['schemas']['HttpHops']
export const METHODS = ['GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'] as const
export const BODY_METHODS = ['POST', 'PUT', 'PATCH', 'DELETE']

const FORBIDDEN = new Set(['connection', 'content-length', 'transfer-encoding', 'upgrade', 'te', 'trailer', 'proxy-connection', 'keep-alive', 'expect'])

/** "Name: value" per line → object; returns the offending line on error. */
export function parseHeaderLines(text: string): { headers: Record<string, string> } | { error: string } {
  const headers: Record<string, string> = {}
  for (const raw of text.split('\n')) {
    const line = raw.trim()
    if (!line)
      continue
    const i = line.indexOf(':')
    const name = i > 0 ? line.slice(0, i).trim() : ''
    if (!/^[!#$%&'*+\-.^\w`|~]+$/.test(name))
      return { error: line }
    if (FORBIDDEN.has(name.toLowerCase()))
      return { error: line }
    headers[name] = line.slice(i + 1).trim()
  }
  return Object.keys(headers).length > 30 ? { error: '> 30 headers' } : { headers }
}
