import { formatIP, parseIP } from '@/utils/ip'
import { classify } from '@/utils/special-ranges'

export const STUN_SERVERS = ['stun:stun.l.google.com:19302', 'stun:stun.cloudflare.com:3478']

export interface Candidate {
  type: string // host, srflx, prflx, relay
  protocol: string
  address: string
  port: number
  kind: 'mdns' | 'private' | 'public' | 'unknown'
}

/** Parses an ICE candidate line (RFC 8839 §5.1). */
export function parseCandidate(line: string): Candidate | undefined {
  const parts = line.replace(/^a=/, '').replace(/^candidate:/, '').trim().split(/\s+/)
  if (parts.length < 8 || parts[6] !== 'typ')
    return undefined
  const address = parts[4]!
  let kind: Candidate['kind'] = 'unknown'
  if (address.endsWith('.local')) {
    kind = 'mdns'
  }
  else {
    const ip = parseIP(address)
    if (ip)
      kind = classify(ip).bogon ? 'private' : 'public'
  }
  const ip = parseIP(address)
  return { type: parts[7]!, protocol: parts[2]!.toLowerCase(), address: ip ? formatIP(ip.version, ip.value) : address, port: Number(parts[5]), kind }
}

export interface Verdict {
  localLeak: boolean
  publicAddresses: string[]
  /** Public addresses from STUN that differ from what websites see over HTTP (e.g. VPN bypass). */
  mismatched: string[]
}

export function verdict(cands: Candidate[], httpIps: string[]): Verdict {
  const publicAddresses = [...new Set(cands.filter(c => c.kind === 'public').map(c => c.address))]
  return {
    localLeak: cands.some(c => c.type === 'host' && c.kind === 'private'),
    publicAddresses,
    mismatched: httpIps.length ? publicAddresses.filter(a => !httpIps.includes(a)) : [],
  }
}
