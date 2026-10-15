import type { components } from '@/api/schema'

export type OpenResolverResult = components['schemas']['OpenResolverResult']
export type Verdict = 'open' | 'closed' | 'silent'

export const verdict = (r: OpenResolverResult): Verdict => (!r.responds ? 'silent' : r.open ? 'open' : 'closed')

/** IP address or hostname, optional port. */
export const isServer = (s: string) => /^(?:[\w.-]+|\[[\da-f:]+\]|[\da-f:]+)(?::\d{1,5})?$/i.test(s.trim())
