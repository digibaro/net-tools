import type { components } from '@/api/schema'

export type TorCheck = components['schemas']['TorCheck']
export { isIpAddress } from '../reverse-dns/reverse-dns.service'

/** Exit verdict: listed exit, relay that is not an exit, or not Tor at all. */
export function verdict(r: TorCheck): 'exit' | 'relay' | 'none' {
  if (r.exitNode)
    return 'exit'
  return r.relays.length ? 'relay' : 'none'
}
