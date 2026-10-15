/** Codec bit rates in kbit/s (payload only). */
export const CODECS = { 'G.711': 64, 'G.722': 64, 'G.729': 8, 'G.726-32': 32, 'iLBC-30': 13.33, 'Opus-24': 24, 'Opus-48': 48, 'AMR-WB-23.85': 23.85 } as const
export type Codec = keyof typeof CODECS

/** Layer 2 framing per packet in bytes (Ethernet incl. preamble and inter-frame gap = 38). */
export const LINKS = { 'ethernet': 38, 'ethernet-no-phy': 18, 'ppp': 7, 'none': 0 } as const
export type Link = keyof typeof LINKS

export interface VoipOptions { codec: Codec, ptimeMs: number, ipv6: boolean, srtp: boolean, vlan: boolean, link: Link }
export interface VoipResult { payloadBytes: number, packetBytes: number, pps: number, kbps: number, overheadPct: number }

export function perCall(o: VoipOptions): VoipResult {
  const payloadBytes = Math.ceil(CODECS[o.codec] * 1000 / 8 * o.ptimeMs / 1000)
  // RTP 12 + UDP 8 + IP 20/40, SRTP auth tag 10, 802.1Q 4.
  const headers = 12 + 8 + (o.ipv6 ? 40 : 20) + (o.srtp ? 10 : 0) + (o.vlan ? 4 : 0) + LINKS[o.link]
  const packetBytes = payloadBytes + headers
  const pps = 1000 / o.ptimeMs
  return { payloadBytes, packetBytes, pps, kbps: packetBytes * 8 * pps / 1000, overheadPct: headers / packetBytes * 100 }
}

/** Erlang B blocking probability for traffic E (erlangs) on N channels (stable recursion). */
export function erlangB(e: number, n: number): number {
  let b = 1
  for (let k = 1; k <= n; k++)
    b = e * b / (k + e * b)
  return b
}

/** Fewest channels so that blocking ≤ target. */
export function channelsFor(e: number, target: number, max = 10000): number {
  let b = 1
  for (let n = 1; n <= max; n++) {
    b = e * b / (n + e * b)
    if (b <= target)
      return n
  }
  return max
}

/** Busy-hour traffic: calls per hour × average minutes / 60. */
export const erlangs = (callsPerHour: number, avgMinutes: number) => callsPerHour * avgMinutes / 60
