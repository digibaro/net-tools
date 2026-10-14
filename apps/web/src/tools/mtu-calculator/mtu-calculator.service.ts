/** Bytes each encapsulation adds inside the outer (link) MTU, with how it adds up. */
export interface Encapsulation {
  id: string
  overhead: number
  breakdown: string
  /** Can be stacked several times (e.g. MPLS labels). */
  stackable?: boolean
}

export const ENCAPSULATIONS: Encapsulation[] = [
  { id: 'vlan', overhead: 4, breakdown: '802.1Q tag 4', stackable: true },
  { id: 'qinq', overhead: 8, breakdown: '2 × 802.1Q tag 4' },
  { id: 'mpls', overhead: 4, breakdown: 'label 4', stackable: true },
  { id: 'pppoe', overhead: 8, breakdown: 'PPPoE 6 + PPP 2' },
  { id: 'gre', overhead: 24, breakdown: 'IPv4 20 + GRE 4' },
  { id: 'gre-key', overhead: 28, breakdown: 'IPv4 20 + GRE 4 + key 4' },
  { id: 'ipip', overhead: 20, breakdown: 'IPv4 20 (IP-in-IP / 6in4)' },
  { id: 'ipv6-tunnel', overhead: 40, breakdown: 'IPv6 40 (4in6 / 6in6)' },
  { id: 'ipsec-gcm', overhead: 57, breakdown: 'IPv4 20 + ESP 8 + IV 8 + pad ≤3 + trailer 2 + ICV 16' },
  { id: 'ipsec-cbc', overhead: 73, breakdown: 'IPv4 20 + ESP 8 + IV 16 + pad ≤15 + trailer 2 + ICV 12' },
  { id: 'nat-t', overhead: 8, breakdown: 'UDP 8 (IPsec NAT traversal)' },
  { id: 'wireguard-v4', overhead: 60, breakdown: 'IPv4 20 + UDP 8 + WireGuard 32' },
  { id: 'wireguard-v6', overhead: 80, breakdown: 'IPv6 40 + UDP 8 + WireGuard 32' },
  { id: 'vxlan-v4', overhead: 50, breakdown: 'IPv4 20 + UDP 8 + VXLAN 8 + inner Ethernet 14' },
  { id: 'vxlan-v6', overhead: 70, breakdown: 'IPv6 40 + UDP 8 + VXLAN 8 + inner Ethernet 14' },
  { id: 'geneve-v4', overhead: 50, breakdown: 'IPv4 20 + UDP 8 + Geneve 8 + inner Ethernet 14 (+ options)' },
]

export interface MtuResult {
  totalOverhead: number
  innerMtu: number
  mssIpv4: number
  mssIpv6: number
  /** IPv6 requires a link MTU of at least 1280 (RFC 8200). */
  belowIpv6Minimum: boolean
}

/** `layers` maps encapsulation id → count (0 = unused). */
export function calculateMtu(linkMtu: number, layers: Record<string, number>): MtuResult | undefined {
  if (!Number.isInteger(linkMtu) || linkMtu < 68 || linkMtu > 65_535)
    return undefined
  let totalOverhead = 0
  for (const e of ENCAPSULATIONS) {
    const n = Math.max(0, Math.floor(layers[e.id] ?? 0))
    totalOverhead += e.overhead * (e.stackable ? n : Math.min(n, 1))
  }
  const innerMtu = linkMtu - totalOverhead
  if (innerMtu < 68)
    return undefined
  return { totalOverhead, innerMtu, mssIpv4: innerMtu - 40, mssIpv6: innerMtu - 60, belowIpv6Minimum: innerMtu < 1280 }
}

/** Compact "id:count,id:count" for the URL. */
export function encodeLayers(layers: Record<string, number>): string {
  return Object.entries(layers).filter(([, n]) => n > 0).map(([id, n]) => (n === 1 ? id : `${id}:${n}`)).join(',')
}

export function decodeLayers(s: string): Record<string, number> {
  const out: Record<string, number> = {}
  for (const part of s.split(',').filter(Boolean)) {
    const [id, n] = part.split(':')
    if (id && ENCAPSULATIONS.some(e => e.id === id))
      out[id] = Math.min(Math.max(Number(n ?? 1) || 1, 0), 16)
  }
  return out
}
