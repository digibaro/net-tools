import type { Decoded, Start } from '../packet-decoder/packet-decoder.service'
import { decode } from '../packet-decoder/packet-decoder.service'

export const MAX_BYTES = 100 * 1024 * 1024
export const MAX_PACKETS = 200_000

export interface RawPacket { ts: number, len: number, data: Uint8Array, linktype: number }
export interface Packet { no: number, time: number, len: number, src: string, dst: string, protocol: string, info: string, decoded: Decoded }
export interface Capture {
  format: 'pcap' | 'pcapng'
  linktypes: number[]
  packets: Packet[]
  truncated: boolean
  bytes: number
  duration: number
  protocols: { name: string, packets: number }[]
  conversations: { a: string, b: string, packets: number, bytes: number }[]
  dns: { query: string, packets: number }[]
}
export type CaptureError = 'format' | 'too-big' | 'linktype'

/** Link types we decode: Ethernet, raw IP, Linux cooked v1/v2, BSD loopback. */
const LINKS: Record<number, Start | 'null'> = { 1: 'ethernet', 101: 'auto', 228: 'ipv4', 229: 'ipv6', 113: 'sll', 276: 'sll2', 0: 'null' }

function readPcap(v: DataView, out: RawPacket[]): number[] | undefined {
  const magic = v.getUint32(0, false)
  const le = magic === 0xD4C3B2A1 || magic === 0x4D3CB2A1
  const nano = magic === 0xA1B23C4D || magic === 0x4D3CB2A1
  if (!le && magic !== 0xA1B2C3D4 && magic !== 0xA1B23C4D)
    return undefined
  const linktype = v.getUint32(20, le) & 0xFFFF
  let o = 24
  while (o + 16 <= v.byteLength && out.length < MAX_PACKETS) {
    const ts = v.getUint32(o, le) + v.getUint32(o + 4, le) / (nano ? 1e9 : 1e6)
    const incl = v.getUint32(o + 8, le)
    const len = v.getUint32(o + 12, le)
    if (o + 16 + incl > v.byteLength)
      break
    out.push({ ts, len, linktype, data: new Uint8Array(v.buffer, v.byteOffset + o + 16, incl) })
    o += 16 + incl
  }
  return [linktype]
}

function readPcapng(v: DataView, out: RawPacket[]): number[] | undefined {
  if (v.getUint32(0, false) !== 0x0A0D0D0A)
    return undefined
  let le = true
  let o = 0
  const ifaces: { linktype: number, tsres: number }[] = []
  while (o + 12 <= v.byteLength && out.length < MAX_PACKETS) {
    const type = v.getUint32(o, le)
    if (type === 0x0A0D0D0A) { // section header: byte order magic decides the endianness
      le = v.getUint32(o + 8, true) === 0x1A2B3C4D
      ifaces.length = 0
    }
    const blen = v.getUint32(o + 4, le)
    if (blen < 12 || o + blen > v.byteLength)
      break
    if (type === 1) { // interface description
      let tsres = 1e-6
      let p = o + 16
      while (p + 4 <= o + blen - 4) { // options: if_tsresol (9)
        const code = v.getUint16(p, le)
        const olen = v.getUint16(p + 2, le)
        if (code === 0)
          break
        if (code === 9 && olen >= 1) {
          const r = v.getUint8(p + 4)
          tsres = r & 0x80 ? 2 ** -(r & 0x7F) : 10 ** -r
        }
        p += 4 + Math.ceil(olen / 4) * 4
      }
      ifaces.push({ linktype: v.getUint16(o + 8, le), tsres })
    }
    else if (type === 6) { // enhanced packet
      const iface = ifaces[v.getUint32(o + 8, le)] ?? { linktype: 1, tsres: 1e-6 }
      const ts = (v.getUint32(o + 12, le) * 2 ** 32 + v.getUint32(o + 16, le)) * iface.tsres
      const incl = v.getUint32(o + 20, le)
      out.push({ ts, len: v.getUint32(o + 24, le), linktype: iface.linktype, data: new Uint8Array(v.buffer, v.byteOffset + o + 28, Math.min(incl, blen - 32)) })
    }
    else if (type === 3) { // simple packet
      const iface = ifaces[0] ?? { linktype: 1, tsres: 1e-6 }
      const len = v.getUint32(o + 8, le)
      out.push({ ts: 0, len, linktype: iface.linktype, data: new Uint8Array(v.buffer, v.byteOffset + o + 12, Math.min(len, blen - 16)) })
    }
    o += blen
  }
  return [...new Set(ifaces.map(i => i.linktype))]
}

function decodeRaw(p: RawPacket): Decoded {
  const how = LINKS[p.linktype]
  if (how === 'null') // BSD loopback: 4-byte address family, then IP
    return decode(p.data.subarray(4), 'auto')
  return decode(p.data, how ?? 'ethernet')
}

export function readCapture(buf: ArrayBuffer): Capture | CaptureError {
  if (buf.byteLength > MAX_BYTES)
    return 'too-big'
  if (buf.byteLength < 24)
    return 'format'
  const v = new DataView(buf)
  const raw: RawPacket[] = []
  const pcapLinks = readPcap(v, raw)
  const links = pcapLinks ?? readPcapng(v, raw)
  if (!links)
    return 'format'
  if (links.length && links.every(l => !(l in LINKS)))
    return 'linktype'
  const t0 = raw[0]?.ts ?? 0
  const protocols = new Map<string, number>()
  const convs = new Map<string, { a: string, b: string, packets: number, bytes: number }>()
  const dns = new Map<string, number>()
  let bytes = 0
  const packets = raw.map((p, i) => {
    const d = decodeRaw(p)
    bytes += p.len
    const proto = d.protocol || 'other'
    protocols.set(proto, (protocols.get(proto) ?? 0) + 1)
    if (d.src && d.dst) {
      const [a, b] = [d.src, d.dst].sort()
      const key = `${a}|${b}`
      const c = convs.get(key) ?? { a: a!, b: b!, packets: 0, bytes: 0 }
      c.packets++
      c.bytes += p.len
      convs.set(key, c)
    }
    const q = d.layers.find(l => l.name === 'DNS')?.fields.filter(f => /^Question \d/.test(f.name)).map(f => f.value)
    for (const name of q ?? [])
      dns.set(name, (dns.get(name) ?? 0) + 1)
    return { no: i + 1, time: p.ts - t0, len: p.len, src: d.src, dst: d.dst, protocol: proto, info: d.info, decoded: d }
  })
  return {
    format: pcapLinks ? 'pcap' : 'pcapng',
    linktypes: links,
    packets,
    truncated: raw.length >= MAX_PACKETS,
    bytes,
    duration: raw.length ? raw.at(-1)!.ts - t0 : 0,
    protocols: [...protocols].map(([name, n]) => ({ name, packets: n })).sort((x, y) => y.packets - x.packets),
    conversations: [...convs.values()].sort((x, y) => y.bytes - x.bytes),
    dns: [...dns].map(([query, n]) => ({ query, packets: n })).sort((x, y) => y.packets - x.packets),
  }
}

/** Builds a classic little-endian pcap (tests and examples). */
export function writePcap(packets: { ts: number, data: Uint8Array }[], linktype = 1): ArrayBuffer {
  const size = 24 + packets.reduce((a, p) => a + 16 + p.data.length, 0)
  const v = new DataView(new ArrayBuffer(size))
  v.setUint32(0, 0xA1B2C3D4, true)
  v.setUint16(4, 2, true)
  v.setUint16(6, 4, true)
  v.setUint32(16, 65535, true)
  v.setUint32(20, linktype, true)
  let o = 24
  for (const p of packets) {
    v.setUint32(o, Math.floor(p.ts), true)
    v.setUint32(o + 4, Math.round((p.ts % 1) * 1e6), true)
    v.setUint32(o + 8, p.data.length, true)
    v.setUint32(o + 12, p.data.length, true)
    new Uint8Array(v.buffer, o + 16, p.data.length).set(p.data)
    o += 16 + p.data.length
  }
  return v.buffer
}
