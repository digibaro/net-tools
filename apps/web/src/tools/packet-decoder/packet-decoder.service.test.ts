import { describe, expect, it } from 'vitest'
import { checksum, decode, parseHex } from './packet-decoder.service'

// Ethernet + IPv4 + UDP + DNS query "example.com A"
const DNS_QUERY = '0011223344550a0b0c0d0e0f08004500003a1c4600004011'
  + '0000c0a80164080808080d2c0035002600001a2b01000001000000000000076578616d706c6503636f6d0000010001'
// Ethernet + VLAN 100 + IPv4 + TCP SYN to 443
const TCP_SYN = 'ffeeddccbbaa11223344556681000064080045000028abcd40004006'
  + '0000c0a80102c633640a3039'
  + '01bb0000000100000000500272100000' + '0000'

function fixIpChecksum(hex: string, ipOffset: number): Uint8Array {
  const b = parseHex(hex)!
  b[ipOffset + 10] = 0
  b[ipOffset + 11] = 0
  const c = checksum(b.subarray(ipOffset, ipOffset + 20))
  b[ipOffset + 10] = c >> 8
  b[ipOffset + 11] = c & 0xFF
  return b
}

describe('packet-decoder', () => {
  it('parses hex dumps in several formats', () => {
    expect(parseHex('45 00 00 1c')).toEqual(new Uint8Array([0x45, 0, 0, 0x1C]))
    expect(parseHex('0000  45 00 00 1c 00 01   E....\n0010  ff ee')).toEqual(new Uint8Array([0x45, 0, 0, 0x1C, 0, 1, 0xFF, 0xEE]))
    expect(parseHex('00000000: 4500 001c  E...')).toEqual(new Uint8Array([0x45, 0, 0, 0x1C]))
    expect(parseHex('abc')).toBeUndefined()
  })
  it('decodes Ethernet / IPv4 / UDP / DNS', () => {
    const d = decode(fixIpChecksum(DNS_QUERY, 14))
    expect(d.layers.map(l => l.name)).toEqual(['Ethernet II', 'IPv4', 'UDP', 'DNS'])
    expect(d.src).toBe('192.168.1.100')
    expect(d.protocol).toBe('DNS')
    expect(d.info).toContain('example.com. A')
    expect(d.layers[1]!.fields.find(f => f.name === 'Header checksum')!.value).toContain('correct')
  })
  it('decodes VLAN tags and TCP flags', () => {
    const d = decode(fixIpChecksum(TCP_SYN, 18))
    expect(d.layers[0]!.fields.find(f => f.name === '802.1Q VLAN')!.value).toBe('100 (priority 0)')
    expect(d.layers.at(-1)!.summary).toBe('12345 → 443 [SYN] (HTTPS)')
  })
  it('reports truncation instead of failing', () => {
    const d = decode(parseHex('450000280000')!, 'ipv4')
    expect(d.error).toMatch(/truncated/)
  })
})
