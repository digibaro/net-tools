import type { Settings } from './wireguard-config.service'
import { describe, expect, it } from 'vitest'
import { buildConfigs, generateKeyPair, generateKeys, publicKey, validate } from './wireguard-config.service'

const hex = (s: string) => Uint8Array.from(s.match(/../g)!.map(b => Number.parseInt(b, 16)))
const b64 = (b: Uint8Array) => btoa(String.fromCharCode(...b))

describe('keys', () => {
  it('matches the RFC 7748 §6.1 vector', () => {
    const priv = b64(hex('77076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a'))
    expect(publicKey(priv)).toBe(b64(hex('8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a')))
    expect(publicKey('short')).toBeUndefined()
  })
  it('clamps generated keys like wg genkey', () => {
    const k = generateKeyPair(n => new Uint8Array(n).fill(0xFF))
    const raw = Uint8Array.from(atob(k.privateKey), c => c.charCodeAt(0))
    expect(raw[0]! & 7).toBe(0)
    expect(raw[31]! & 0xC0).toBe(0x40)
    expect(publicKey(k.privateKey)).toBe(k.publicKey)
  })
})

const settings: Settings = { endpoint: 'vpn.example.com', listenPort: 51820, address4: '10.8.0.1/24', address6: 'fd42::1/64', dns: '1.1.1.1', mtu: null, peers: 2, routing: 'full', allowedIps: '', keepalive: 25, presharedKeys: true }

describe('configs', () => {
  it('builds server and peer configs', () => {
    const keys = generateKeys(2)
    const c = buildConfigs(settings, keys)
    expect(c.server).toContain('Address = 10.8.0.1/24, fd42::1/64')
    expect(c.server).toContain(`PublicKey = ${keys.peers[1]!.publicKey}`)
    expect(c.server).toContain('AllowedIPs = 10.8.0.3/32, fd42::3/128')
    expect(c.peers[0]!.config).toContain('Endpoint = vpn.example.com:51820')
    expect(c.peers[0]!.config).toContain('AllowedIPs = 0.0.0.0/0, ::/0')
    expect(c.peers[0]!.config).toContain(`PresharedKey = ${keys.peers[0]!.psk}`)
    expect(buildConfigs({ ...settings, routing: 'split', address6: '' }, keys).peers[1]!.config).toContain('AllowedIPs = 10.8.0.0/24')
  })
  it('validates settings', () => {
    expect(validate(settings)).toBeUndefined()
    expect(validate({ ...settings, endpoint: 'bad host' })).toBe('endpoint')
    expect(validate({ ...settings, endpoint: '[2001:db8::1]:51820' })).toBeUndefined()
    expect(validate({ ...settings, address4: '10.8.0.1/31' })).toBe('address4')
    expect(validate({ ...settings, address4: '10.8.0.1/29', peers: 6 })).toBe('full')
    expect(validate({ ...settings, dns: '1.1.1' })).toBe('dns')
    expect(validate({ ...settings, routing: 'custom', allowedIps: '' })).toBe('allowedIps')
  })
})
