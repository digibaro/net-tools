import type { components } from '@/api/schema'

export type Whois = components['schemas']['Whois']
export { isRegistryQuery } from '@/utils/registry-query'

/** The most specific answer: the last response in the referral chain that has text. */
export function finalResponse(w: Whois) {
  return [...w.chain].reverse().find(r => r.text) ?? w.chain.at(-1)
}
