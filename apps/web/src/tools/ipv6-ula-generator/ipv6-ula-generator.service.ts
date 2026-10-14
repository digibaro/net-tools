import { formatIPv6 } from '@/utils/ip'

const NTP_EPOCH_OFFSET = 2_208_988_800n // seconds between 1900-01-01 and 1970-01-01

/** 64-bit NTP timestamp (RFC 5905) for a JS time in ms. */
export function ntpTimestamp(ms: number): bigint {
  const seconds = BigInt(Math.floor(ms / 1000)) + NTP_EPOCH_OFFSET
  const fraction = (BigInt(ms % 1000) << 32n) / 1000n
  return (seconds << 32n) | fraction
}

function toBytes(v: bigint, n: number): Uint8Array<ArrayBuffer> {
  const out = new Uint8Array(n)
  for (let i = n - 1; i >= 0; i--) {
    out[i] = Number(v & 0xFFn)
    v >>= 8n
  }
  return out
}

/** RFC 4193 §3.2.2: SHA-1(NTP time ‖ EUI-64), least significant 40 bits. */
export async function computeGlobalId(ntp: bigint, eui64: bigint): Promise<bigint> {
  const input = new Uint8Array(16)
  input.set(toBytes(ntp, 8), 0)
  input.set(toBytes(eui64, 8), 8)
  const digest = new Uint8Array(await crypto.subtle.digest('SHA-1', input))
  return [...digest.slice(15)].reduce((acc, b) => (acc << 8n) | BigInt(b), 0n)
}

export function randomEui64(): bigint {
  const b = crypto.getRandomValues(new Uint8Array(8))
  return [...b].reduce((acc, x) => (acc << 8n) | BigInt(x), 0n)
}

export function parseGlobalId(s: string): bigint | undefined {
  return /^[0-9a-f]{10}$/i.test(s.trim()) ? BigInt(`0x${s.trim()}`) : undefined
}

export interface Ula {
  globalId: string
  prefix: string
  firstSubnet: string
  lastSubnet: string
  exampleSubnets: string[]
  subnetCount: string
}

export function ulaFromGlobalId(globalId: bigint): Ula {
  const base = (0xFDn << 120n) | (globalId << 80n)
  const subnet = (id: bigint) => `${formatIPv6(base | (id << 64n))}/64`
  return {
    globalId: globalId.toString(16).padStart(10, '0'),
    prefix: `${formatIPv6(base)}/48`,
    firstSubnet: subnet(0n),
    lastSubnet: subnet(0xFFFFn),
    exampleSubnets: [1n, 2n, 3n, 0x10n].map(subnet),
    subnetCount: '65,536',
  }
}
