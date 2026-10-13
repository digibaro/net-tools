import { describe, expect, it } from 'vitest'
import { parseCidr, parseIP } from '@/utils/ip'
import { generate, randomBelow } from './random-ip-generator.service'

describe('randomBelow', () => {
  it('stays in range and rejects out-of-range draws', () => {
    // First draw 0xff (masked to 7 > 5) is rejected, then 3 is accepted.
    const seq = [0xFF, 0x03]
    const bytes = (n: number) => Uint8Array.from(seq.splice(0, n))
    expect(randomBelow(6n, bytes)).toBe(3n)
    for (let i = 0; i < 200; i++) {
      const v = randomBelow(1000n)
      expect(v >= 0n && v < 1000n).toBe(true)
    }
  })
})

describe('generate', () => {
  it('generates addresses inside the prefix', () => {
    const net = parseCidr('10.20.0.0/16')!
    const r = generate('10.20.0.0/16', { count: 100, unique: true, usableOnly: true }) as string[]
    expect(r).toHaveLength(100)
    expect(new Set(r).size).toBe(100)
    for (const ip of r)
      expect((parseIP(ip)!.value >> 16n) << 16n).toBe(net.network)
  })
  it('skips network and broadcast and exhausts small ranges', () => {
    const r = generate('192.0.2.0/30', { count: 2, unique: true, usableOnly: true }) as string[]
    expect(r.sort()).toEqual(['192.0.2.1', '192.0.2.2'])
    expect(generate('192.0.2.0/30', { count: 3, unique: true, usableOnly: true })).toBe('too-small')
    expect((generate('192.0.2.0/30', { count: 5, unique: false, usableOnly: true }) as string[]).length).toBe(5)
  })
  it('handles IPv6 and bad input', () => {
    const r = generate('2001:db8::/64', { count: 3, unique: true, usableOnly: true }) as string[]
    expect(r.every(ip => ip.startsWith('2001:db8::'))).toBe(true)
    expect(generate('nope', { count: 1, unique: false, usableOnly: false })).toBe('network')
  })
})
