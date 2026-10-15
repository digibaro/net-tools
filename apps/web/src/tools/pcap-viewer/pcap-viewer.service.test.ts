import type { Capture } from './pcap-viewer.service'
import { describe, expect, it } from 'vitest'
import { parseHex } from '../packet-decoder/packet-decoder.service'
import { readCapture, writePcap } from './pcap-viewer.service'

const DNS = parseHex('0011223344550a0b0c0d0e0f08004500003a1c46000040110000c0a80164080808080d2c0035002600001a2b01000001000000000000076578616d706c6503636f6d0000010001')!
const SYN = parseHex('ffeeddccbbaa112233445566080045000028abcd400040060000c0a80102c633640a303901bb00000001000000005002721000000000')!

function pcapng(): ArrayBuffer {
  // SHB (28) + IDB (20) + EPB with the SYN packet
  const pad = (n: number) => Math.ceil(n / 4) * 4
  const epbLen = 32 + pad(SYN.length)
  const v = new DataView(new ArrayBuffer(28 + 20 + epbLen))
  v.setUint32(0, 0x0A0D0D0A, true)
  v.setUint32(4, 28, true)
  v.setUint32(8, 0x1A2B3C4D, true)
  v.setUint16(12, 1, true)
  v.setInt32(16, -1, true)
  v.setInt32(20, -1, true)
  v.setUint32(24, 28, true)
  v.setUint32(28, 1, true)
  v.setUint32(32, 20, true)
  v.setUint16(36, 1, true)
  v.setUint32(40, 65535, true)
  v.setUint32(44, 20, true)
  const o = 48
  v.setUint32(o, 6, true)
  v.setUint32(o + 4, epbLen, true)
  v.setUint32(o + 8, 0, true)
  v.setUint32(o + 12, 0, true)
  v.setUint32(o + 16, 1_000_000, true)
  v.setUint32(o + 20, SYN.length, true)
  v.setUint32(o + 24, SYN.length, true)
  new Uint8Array(v.buffer, o + 28, SYN.length).set(SYN)
  v.setUint32(o + epbLen - 4, epbLen, true)
  return v.buffer
}

describe('pcap-viewer', () => {
  it('reads a classic pcap', () => {
    const c = readCapture(writePcap([{ ts: 100, data: DNS }, { ts: 100.5, data: SYN }, { ts: 101, data: DNS }])) as Capture
    expect(c.format).toBe('pcap')
    expect(c.packets).toHaveLength(3)
    expect(c.duration).toBe(1)
    expect(c.packets[1]!.protocol).toBe('HTTPS')
    expect(c.protocols[0]).toEqual({ name: 'DNS', packets: 2 })
    expect(c.dns).toEqual([{ query: 'example.com. A', packets: 2 }])
    expect(c.conversations[0]!.packets).toBe(2)
  })
  it('reads pcapng', () => {
    const c = readCapture(pcapng()) as Capture
    expect(c.format).toBe('pcapng')
    expect(c.packets[0]!.info).toContain('[SYN]')
    expect(c.packets[0]!.dst).toBe('198.51.100.10')
  })
  it('rejects other files', () => {
    expect(readCapture(new ArrayBuffer(100))).toBe('format')
    expect(readCapture(writePcap([], 105))).toBe('linktype')
  })
})
