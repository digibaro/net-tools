import { formatIPv4, formatIPv6 } from '@/utils/ip'

export interface Field { name: string, value: string }
export interface Layer { name: string, summary: string, offset: number, length: number, fields: Field[] }
export interface Decoded { layers: Layer[], src: string, dst: string, protocol: string, info: string, error?: string }
export type Start = 'auto' | 'ethernet' | 'ipv4' | 'ipv6' | 'sll' | 'sll2'

/** Accepts plain hex, Wireshark "Copy as hex stream", hexdump -C / xxd / tcpdump -XX lines with offsets and ASCII. */
export function parseHex(input: string): Uint8Array | undefined {
  const bytes: string[] = []
  for (const line of input.split('\n')) {
    const tokens = line.replace(/\|.*\|/, ' ').trim().split(/\s+/).filter(Boolean)
    // A leading offset ("0010", "00000010:", "0x0010:") is followed by byte groups.
    if (tokens.length > 1 && (/^(?:0x)?[\da-f]+:$/i.test(tokens[0]!) || (/^[\da-f]{4,8}$/i.test(tokens[0]!) && /^[\da-f]{2}$/i.test(tokens[1]!))))
      tokens.shift()
    for (const tok of tokens) {
      const h = tok.replace(/^0x/i, '')
      if (!/^(?:[\da-f]{2})+$/i.test(h))
        break // the ASCII column starts here
      bytes.push(...h.match(/../g)!)
    }
  }
  return bytes.length ? Uint8Array.from(bytes, b => Number.parseInt(b, 16)) : undefined
}

const ETHERTYPES: Record<number, string> = { 0x0800: 'IPv4', 0x86DD: 'IPv6', 0x0806: 'ARP', 0x8100: '802.1Q', 0x88A8: '802.1ad', 0x88CC: 'LLDP', 0x8847: 'MPLS' }
const IPPROTO: Record<number, string> = { 1: 'ICMP', 2: 'IGMP', 6: 'TCP', 17: 'UDP', 41: 'IPv6', 47: 'GRE', 50: 'ESP', 51: 'AH', 58: 'ICMPv6', 89: 'OSPF', 103: 'PIM', 112: 'VRRP', 132: 'SCTP' }
const PORTS: Record<number, string> = { 20: 'FTP-data', 21: 'FTP', 22: 'SSH', 23: 'Telnet', 25: 'SMTP', 53: 'DNS', 67: 'DHCP', 68: 'DHCP', 80: 'HTTP', 110: 'POP3', 123: 'NTP', 143: 'IMAP', 161: 'SNMP', 179: 'BGP', 389: 'LDAP', 443: 'HTTPS', 500: 'IKE', 514: 'Syslog', 546: 'DHCPv6', 547: 'DHCPv6', 587: 'Submission', 853: 'DoT', 993: 'IMAPS', 1812: 'RADIUS', 3389: 'RDP', 4500: 'IPsec NAT-T', 5060: 'SIP', 5353: 'mDNS' }
const ICMP4: Record<number, string> = { 0: 'Echo reply', 3: 'Destination unreachable', 5: 'Redirect', 8: 'Echo request', 11: 'Time exceeded', 12: 'Parameter problem' }
const ICMP6: Record<number, string> = { 1: 'Destination unreachable', 2: 'Packet too big', 3: 'Time exceeded', 128: 'Echo request', 129: 'Echo reply', 133: 'Router solicitation', 134: 'Router advertisement', 135: 'Neighbor solicitation', 136: 'Neighbor advertisement' }
const DNS_TYPES: Record<number, string> = { 1: 'A', 2: 'NS', 5: 'CNAME', 6: 'SOA', 12: 'PTR', 15: 'MX', 16: 'TXT', 28: 'AAAA', 33: 'SRV', 41: 'OPT', 43: 'DS', 46: 'RRSIG', 48: 'DNSKEY', 64: 'SVCB', 65: 'HTTPS', 255: 'ANY' }

const hex2 = (n: number, w = 4) => `0x${n.toString(16).padStart(w, '0')}`
const mac = (b: Uint8Array, o: number) => Array.from(b.subarray(o, o + 6), x => x.toString(16).padStart(2, '0')).join(':')
const u16 = (b: Uint8Array, o: number) => (b[o]! << 8) | b[o + 1]!
const u32 = (b: Uint8Array, o: number) => ((b[o]! << 24) >>> 0) + (b[o + 1]! << 16) + (b[o + 2]! << 8) + b[o + 3]!
const ip4 = (b: Uint8Array, o: number) => formatIPv4(BigInt(u32(b, o)))
const ip6 = (b: Uint8Array, o: number) => formatIPv6(Array.from(b.subarray(o, o + 16)).reduce((a, x) => (a << 8n) | BigInt(x), 0n))

class Short extends Error {}
function need(b: Uint8Array, o: number, n: number) {
  if (o + n > b.length)
    throw new Short(`truncated at byte ${b.length} (needed ${o + n})`)
}

/** Internet checksum (RFC 1071) over the given bytes; 0 means the stored checksum is correct. */
export function checksum(b: Uint8Array): number {
  let sum = 0
  for (let i = 0; i < b.length; i += 2)
    sum += (b[i]! << 8) + (b[i + 1] ?? 0)
  while (sum > 0xFFFF)
    sum = (sum & 0xFFFF) + (sum >>> 16)
  return ~sum & 0xFFFF
}

function dnsName(b: Uint8Array, o: number, base: number): [string, number] {
  const labels: string[] = []
  let pos = o
  let end = -1
  for (let jumps = 0; jumps < 20; jumps++) {
    need(b, pos, 1)
    const len = b[pos]!
    if (len === 0) {
      pos++
      break
    }
    if ((len & 0xC0) === 0xC0) {
      need(b, pos, 2)
      if (end < 0)
        end = pos + 2
      pos = base + (((len & 0x3F) << 8) | b[pos + 1]!)
      continue
    }
    need(b, pos + 1, len)
    labels.push(String.fromCharCode(...b.subarray(pos + 1, pos + 1 + len)))
    pos += 1 + len
  }
  return [`${labels.join('.')}.`, end < 0 ? pos : end]
}

function decodeDns(b: Uint8Array, o: number): Layer {
  need(b, o, 12)
  const flags = u16(b, o + 2)
  const qd = u16(b, o + 4)
  const an = u16(b, o + 6)
  const qr = flags >> 15 ? 'response' : 'query'
  const rcode = flags & 0xF
  const fields: Field[] = [
    { name: 'Transaction ID', value: hex2(u16(b, o)) },
    { name: 'Flags', value: `${hex2(flags)} (${qr}${flags & 0x0100 ? ', RD' : ''}${flags & 0x0080 ? ', RA' : ''}${flags & 0x0400 ? ', AA' : ''}${flags & 0x0200 ? ', TC' : ''}, rcode ${rcode})` },
    { name: 'Questions / answers / authority / additional', value: `${qd} / ${an} / ${u16(b, o + 8)} / ${u16(b, o + 10)}` },
  ]
  let pos = o + 12
  const qs: string[] = []
  for (let i = 0; i < Math.min(qd, 10); i++) {
    const [name, next] = dnsName(b, pos, o)
    need(b, next, 4)
    const type = DNS_TYPES[u16(b, next)] ?? String(u16(b, next))
    qs.push(`${name} ${type}`)
    fields.push({ name: `Question ${i + 1}`, value: `${name} ${type}` })
    pos = next + 4
  }
  return { name: 'DNS', summary: `${qr} ${qs.join(', ')}${qr === 'response' ? ` → ${an} answer(s), rcode ${rcode}` : ''}`, offset: o, length: b.length - o, fields }
}

function decodeL4(b: Uint8Array, o: number, proto: number, v6: boolean, d: Decoded) {
  if (proto === 6) {
    need(b, o, 20)
    const sp = u16(b, o)
    const dp = u16(b, o + 2)
    const off = (b[o + 12]! >> 4) * 4
    const f = b[o + 13]!
    const names = ['FIN', 'SYN', 'RST', 'PSH', 'ACK', 'URG', 'ECE', 'CWR'].filter((_, i) => f & (1 << i))
    const svc = PORTS[dp] ?? PORTS[sp]
    d.layers.push({ name: 'TCP', summary: `${sp} → ${dp} [${names.join(', ')}]${svc ? ` (${svc})` : ''}`, offset: o, length: off, fields: [
      { name: 'Source port', value: String(sp) },
      { name: 'Destination port', value: String(dp) },
      { name: 'Sequence number', value: String(u32(b, o + 4)) },
      { name: 'Acknowledgment number', value: String(u32(b, o + 8)) },
      { name: 'Header length', value: `${off} bytes` },
      { name: 'Flags', value: names.join(', ') || 'none' },
      { name: 'Window', value: String(u16(b, o + 14)) },
      { name: 'Checksum', value: hex2(u16(b, o + 16)) },
      { name: 'Payload', value: `${Math.max(0, b.length - o - off)} bytes` },
    ] })
    d.protocol = svc ?? 'TCP'
    d.info = `${sp} → ${dp} [${names.join(', ')}] Seq=${u32(b, o + 4)} Win=${u16(b, o + 14)}`
    return
  }
  if (proto === 17) {
    need(b, o, 8)
    const sp = u16(b, o)
    const dp = u16(b, o + 2)
    const svc = PORTS[dp] ?? PORTS[sp]
    d.layers.push({ name: 'UDP', summary: `${sp} → ${dp}${svc ? ` (${svc})` : ''}`, offset: o, length: 8, fields: [
      { name: 'Source port', value: String(sp) },
      { name: 'Destination port', value: String(dp) },
      { name: 'Length', value: String(u16(b, o + 4)) },
      { name: 'Checksum', value: hex2(u16(b, o + 6)) },
    ] })
    d.protocol = svc ?? 'UDP'
    d.info = `${sp} → ${dp} Len=${u16(b, o + 4) - 8}`
    if ([53, 5353].includes(sp) || [53, 5353].includes(dp)) {
      const dns = decodeDns(b, o + 8)
      d.layers.push(dns)
      d.protocol = sp === 5353 || dp === 5353 ? 'mDNS' : 'DNS'
      d.info = dns.summary
    }
    return
  }
  if (proto === 1 || proto === 58) {
    need(b, o, 4)
    const type = b[o]!
    const name = (v6 ? ICMP6 : ICMP4)[type] ?? `type ${type}`
    d.layers.push({ name: v6 ? 'ICMPv6' : 'ICMP', summary: `${name} (code ${b[o + 1]})`, offset: o, length: b.length - o, fields: [
      { name: 'Type', value: `${type} (${name})` },
      { name: 'Code', value: String(b[o + 1]) },
      { name: 'Checksum', value: hex2(u16(b, o + 2)) },
      ...(type === 8 || type === 0 || type === 128 || type === 129 ? [{ name: 'Identifier / sequence', value: `${u16(b, o + 4)} / ${u16(b, o + 6)}` }] : []),
    ] })
    d.protocol = v6 ? 'ICMPv6' : 'ICMP'
    d.info = name
  }
}

function decodeIPv4(b: Uint8Array, o: number, d: Decoded) {
  need(b, o, 20)
  const ihl = (b[o]! & 0xF) * 4
  need(b, o, ihl)
  const proto = b[o + 9]!
  const fragOff = u16(b, o + 6)
  const ok = checksum(b.subarray(o, o + ihl)) === 0
  d.src = ip4(b, o + 12)
  d.dst = ip4(b, o + 16)
  d.layers.push({ name: 'IPv4', summary: `${d.src} → ${d.dst}, ${IPPROTO[proto] ?? proto}, TTL ${b[o + 8]}`, offset: o, length: ihl, fields: [
    { name: 'Version / header length', value: `${b[o]! >> 4} / ${ihl} bytes` },
    { name: 'DSCP / ECN', value: `${b[o + 1]! >> 2} / ${b[o + 1]! & 3}` },
    { name: 'Total length', value: String(u16(b, o + 2)) },
    { name: 'Identification', value: hex2(u16(b, o + 4)) },
    { name: 'Flags', value: [fragOff & 0x4000 ? 'DF' : '', fragOff & 0x2000 ? 'MF' : ''].filter(Boolean).join(', ') || 'none' },
    { name: 'Fragment offset', value: String((fragOff & 0x1FFF) * 8) },
    { name: 'TTL', value: String(b[o + 8]) },
    { name: 'Protocol', value: `${proto} (${IPPROTO[proto] ?? '?'})` },
    { name: 'Header checksum', value: `${hex2(u16(b, o + 10))} (${ok ? 'correct' : 'incorrect'})` },
    { name: 'Source', value: d.src },
    { name: 'Destination', value: d.dst },
  ] })
  d.protocol = IPPROTO[proto] ?? `IP ${proto}`
  if ((fragOff & 0x1FFF) === 0)
    decodeL4(b, o + ihl, proto, false, d)
}

function decodeIPv6(b: Uint8Array, o: number, d: Decoded) {
  need(b, o, 40)
  let next = b[o + 6]!
  d.src = ip6(b, o + 8)
  d.dst = ip6(b, o + 24)
  const vtf = u32(b, o)
  d.layers.push({ name: 'IPv6', summary: `${d.src} → ${d.dst}, ${IPPROTO[next] ?? next}, hop limit ${b[o + 7]}`, offset: o, length: 40, fields: [
    { name: 'Traffic class', value: String((vtf >>> 20) & 0xFF) },
    { name: 'Flow label', value: hex2(vtf & 0xFFFFF, 5) },
    { name: 'Payload length', value: String(u16(b, o + 4)) },
    { name: 'Next header', value: `${next} (${IPPROTO[next] ?? '?'})` },
    { name: 'Hop limit', value: String(b[o + 7]) },
    { name: 'Source', value: d.src },
    { name: 'Destination', value: d.dst },
  ] })
  let pos = o + 40
  // Hop-by-hop, routing and destination options: skip to the upper layer.
  for (let i = 0; i < 8 && [0, 43, 60].includes(next); i++) {
    need(b, pos, 2)
    const len = (b[pos + 1]! + 1) * 8
    d.layers.push({ name: `IPv6 ext. header ${next}`, summary: `${len} bytes`, offset: pos, length: len, fields: [] })
    next = b[pos]!
    pos += len
  }
  d.protocol = IPPROTO[next] ?? `IP ${next}`
  decodeL4(b, pos, next, true, d)
}

function decodeArp(b: Uint8Array, o: number, d: Decoded) {
  need(b, o, 28)
  const op = u16(b, o + 6)
  d.src = ip4(b, o + 14)
  d.dst = ip4(b, o + 24)
  d.info = op === 1 ? `Who has ${d.dst}? Tell ${d.src}` : `${d.src} is at ${mac(b, o + 8)}`
  d.protocol = 'ARP'
  d.layers.push({ name: 'ARP', summary: d.info, offset: o, length: 28, fields: [
    { name: 'Operation', value: op === 1 ? 'request' : op === 2 ? 'reply' : String(op) },
    { name: 'Sender MAC / IP', value: `${mac(b, o + 8)} / ${d.src}` },
    { name: 'Target MAC / IP', value: `${mac(b, o + 18)} / ${d.dst}` },
  ] })
}

function byEthertype(b: Uint8Array, o: number, et: number, d: Decoded) {
  if (et === 0x0800)
    decodeIPv4(b, o, d)
  else if (et === 0x86DD)
    decodeIPv6(b, o, d)
  else if (et === 0x0806)
    decodeArp(b, o, d)
  else
    d.protocol = ETHERTYPES[et] ?? hex2(et)
}

function decodeEthernet(b: Uint8Array, o: number, d: Decoded) {
  need(b, o, 14)
  let et = u16(b, o + 12)
  let pos = o + 14
  const fields: Field[] = [{ name: 'Destination', value: mac(b, o) }, { name: 'Source', value: mac(b, o + 6) }]
  while ((et === 0x8100 || et === 0x88A8) && pos + 4 <= b.length) {
    fields.push({ name: et === 0x8100 ? '802.1Q VLAN' : '802.1ad S-VLAN', value: `${u16(b, pos) & 0xFFF} (priority ${u16(b, pos) >> 13})` })
    et = u16(b, pos + 2)
    pos += 4
  }
  fields.push({ name: 'Type', value: `${hex2(et)} (${ETHERTYPES[et] ?? '?'})` })
  d.src = mac(b, o + 6)
  d.dst = mac(b, o)
  d.layers.push({ name: 'Ethernet II', summary: `${d.src} → ${d.dst}, ${ETHERTYPES[et] ?? hex2(et)}`, offset: o, length: pos - o, fields })
  byEthertype(b, pos, et, d)
}

/** Linux cooked capture v1 (16 bytes) and v2 (20 bytes), as tcpdump -i any writes. */
function decodeSll(b: Uint8Array, d: Decoded, v2: boolean) {
  need(b, 0, v2 ? 20 : 16)
  const et = v2 ? u16(b, 0) : u16(b, 14)
  d.layers.push({ name: v2 ? 'Linux cooked v2' : 'Linux cooked', summary: ETHERTYPES[et] ?? hex2(et), offset: 0, length: v2 ? 20 : 16, fields: [{ name: 'Protocol', value: `${hex2(et)} (${ETHERTYPES[et] ?? '?'})` }] })
  byEthertype(b, v2 ? 20 : 16, et, d)
}

export function decode(b: Uint8Array, start: Start = 'auto'): Decoded {
  const d: Decoded = { layers: [], src: '', dst: '', protocol: '', info: '' }
  let s = start
  if (s === 'auto') {
    const v = b[0]! >> 4
    s = v === 4 && (b[0]! & 0xF) >= 5 ? 'ipv4' : v === 6 && b.length >= 40 ? 'ipv6' : 'ethernet'
  }
  try {
    if (s === 'ethernet')
      decodeEthernet(b, 0, d)
    else if (s === 'ipv4')
      decodeIPv4(b, 0, d)
    else if (s === 'ipv6')
      decodeIPv6(b, 0, d)
    else
      decodeSll(b, d, s === 'sll2')
  }
  catch (e) {
    if (!(e instanceof Short))
      throw e
    d.error = e.message
  }
  if (!d.info)
    d.info = d.layers.at(-1)?.summary ?? ''
  return d
}
