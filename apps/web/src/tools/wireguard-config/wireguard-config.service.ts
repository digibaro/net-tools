import { x25519 } from '@noble/curves/ed25519.js'
import { isValidHost } from '@/utils/host'
import { cidrLast, formatIP, parseCidr, parseIP } from '@/utils/ip'

export const MAX_PEERS = 50

const toB64 = (b: Uint8Array) => btoa(String.fromCharCode(...b))

function fromB64(s: string): Uint8Array | undefined {
  try {
    const b = Uint8Array.from(atob(s.trim()), c => c.charCodeAt(0))
    return b.length === 32 ? b : undefined
  }
  catch {
    return undefined
  }
}

export interface KeyPair { privateKey: string, publicKey: string }

/** Like `wg genkey | wg pubkey`: a clamped Curve25519 secret and its public key, base64. */
export function generateKeyPair(random: (n: number) => Uint8Array = n => crypto.getRandomValues(new Uint8Array(n))): KeyPair {
  const k = random(32)
  k[0]! &= 248
  k[31]! &= 127
  k[31]! |= 64
  return { privateKey: toB64(k), publicKey: toB64(x25519.getPublicKey(k)) }
}

/** `wg pubkey`; undefined for anything that is not a 32-byte base64 key. */
export function publicKey(privateKey: string): string | undefined {
  const k = fromB64(privateKey)
  return k ? toB64(x25519.getPublicKey(k)) : undefined
}

/** `wg genpsk`. */
export const generatePsk = () => toB64(crypto.getRandomValues(new Uint8Array(32)))

export interface Settings {
  endpoint: string
  listenPort: number
  /** Server tunnel address with prefix, e.g. 10.8.0.1/24. */
  address4: string
  /** Optional IPv6 tunnel address, e.g. fd42::1/64. */
  address6: string
  dns: string
  mtu: number | null
  peers: number
  /** 'full' routes everything, 'split' only the tunnel networks, 'custom' uses allowedIps. */
  routing: 'full' | 'split' | 'custom'
  allowedIps: string
  keepalive: number
  presharedKeys: boolean
}

export interface Keys { server: KeyPair, peers: (KeyPair & { psk: string })[] }

export type SettingsError = 'endpoint' | 'port' | 'address4' | 'address6' | 'dns' | 'peers' | 'allowedIps' | 'mtu' | 'full'

const list = (s: string) => s.split(/[\s,]+/).map(x => x.trim()).filter(Boolean)

export function validate(s: Settings): SettingsError | undefined {
  const ep = s.endpoint.trim()
  const host = ep.startsWith('[') ? ep.slice(1, ep.indexOf(']')) : ep.replace(/:\d+$/, '')
  if (!host || (!isValidHost(host) && !parseIP(host)))
    return 'endpoint'
  if (!Number.isInteger(s.listenPort) || s.listenPort < 1 || s.listenPort > 65535)
    return 'port'
  const v4 = parseCidr(s.address4, { allowBare: false })
  if (!v4 || v4.version !== 4 || v4.prefix > 30)
    return 'address4'
  if (s.address6.trim()) {
    const v6 = parseCidr(s.address6, { allowBare: false })
    if (!v6 || v6.version !== 6 || v6.prefix > 126)
      return 'address6'
  }
  if (list(s.dns).some(d => !parseIP(d)))
    return 'dns'
  if (s.mtu !== null && (s.mtu < 1280 || s.mtu > 9000))
    return 'mtu'
  if (!Number.isInteger(s.peers) || s.peers < 1 || s.peers > MAX_PEERS)
    return 'peers'
  if (s.routing === 'custom' && (!list(s.allowedIps).length || list(s.allowedIps).some(c => !parseCidr(c, { allowBare: false }))))
    return 'allowedIps'
  // Peers take the addresses after the server's; they must fit in the tunnel network.
  if (v4.address + BigInt(s.peers) >= cidrLast(v4))
    return 'full'
  return undefined
}

function peerAddress(cidr: string, i: number): string | undefined {
  const c = parseCidr(cidr, { allowBare: false })
  if (!c)
    return undefined
  return `${formatIP(c.version, c.address + BigInt(i + 1))}/${c.version === 4 ? 32 : 128}`
}

function network(cidr: string): string {
  const c = parseCidr(cidr, { allowBare: false })!
  return `${formatIP(c.version, c.network)}/${c.prefix}`
}

export interface Configs { server: string, peers: { name: string, address: string, config: string }[] }

export function buildConfigs(s: Settings, keys: Keys): Configs {
  const v6 = s.address6.trim()
  const endpoint = /:\d+$/.test(s.endpoint.trim()) ? s.endpoint.trim() : `${s.endpoint.trim()}:${s.listenPort}`
  const allowed = s.routing === 'full'
    ? ['0.0.0.0/0', ...(v6 ? ['::/0'] : [])]
    : s.routing === 'split'
      ? [network(s.address4), ...(v6 ? [network(v6)] : [])]
      : list(s.allowedIps)
  const server = ['[Interface]', `Address = ${[s.address4.trim(), v6].filter(Boolean).join(', ')}`, `ListenPort = ${s.listenPort}`, `PrivateKey = ${keys.server.privateKey}`]
  if (s.mtu)
    server.push(`MTU = ${s.mtu}`)
  const peers = keys.peers.slice(0, s.peers).map((k, i) => {
    const name = `peer${i + 1}`
    const addrs = [peerAddress(s.address4, i), v6 ? peerAddress(v6, i) : undefined].filter(Boolean) as string[]
    server.push('', '[Peer]', `# ${name}`, `PublicKey = ${k.publicKey}`, ...(s.presharedKeys ? [`PresharedKey = ${k.psk}`] : []), `AllowedIPs = ${addrs.join(', ')}`)
    const client = ['[Interface]', `PrivateKey = ${k.privateKey}`, `Address = ${addrs.join(', ')}`]
    if (list(s.dns).length)
      client.push(`DNS = ${list(s.dns).join(', ')}`)
    if (s.mtu)
      client.push(`MTU = ${s.mtu}`)
    client.push('', '[Peer]', `PublicKey = ${keys.server.publicKey}`, ...(s.presharedKeys ? [`PresharedKey = ${k.psk}`] : []), `Endpoint = ${endpoint}`, `AllowedIPs = ${allowed.join(', ')}`)
    if (s.keepalive > 0)
      client.push(`PersistentKeepalive = ${s.keepalive}`)
    return { name, address: addrs.join(', '), config: `${client.join('\n')}\n` }
  })
  return { server: `${server.join('\n')}\n`, peers }
}

export function generateKeys(n: number): Keys {
  return { server: generateKeyPair(), peers: Array.from({ length: n }, () => ({ ...generateKeyPair(), psk: generatePsk() })) }
}
