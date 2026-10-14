import { formatIP, parseIP } from '@/utils/ip'
import { classify } from '@/utils/special-ranges'

export const IPIFY = {
  v4: 'https://api.ipify.org?format=json',
  v6: 'https://api6.ipify.org?format=json',
} as const

/**
 * Asks ipify for the public address of one family. Returns undefined when that family has no
 * connectivity (request fails) or the answer is not an address of the expected family.
 */
export async function fetchPublicIp(family: 4 | 6, fetchFn: typeof fetch = fetch, timeoutMs = 6000): Promise<string | undefined> {
  try {
    const res = await fetchFn(family === 4 ? IPIFY.v4 : IPIFY.v6, { signal: AbortSignal.timeout(timeoutMs), cache: 'no-store', referrerPolicy: 'no-referrer' })
    if (!res.ok)
      return undefined
    const body: unknown = await res.json()
    const ip = typeof body === 'object' && body && 'ip' in body && typeof body.ip === 'string' ? parseIP(body.ip) : undefined
    return ip?.version === family ? formatIP(ip.version, ip.value) : undefined
  }
  catch {
    return undefined
  }
}

export type Relation = 'same' | 'private' | 'different'

/** How the address the server saw relates to the public addresses. */
export function relation(serverIp: string, publicIps: (string | undefined)[]): Relation {
  if (publicIps.includes(serverIp))
    return 'same'
  const ip = parseIP(serverIp)
  return ip && classify(ip).bogon ? 'private' : 'different'
}
