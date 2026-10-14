import { bitLength, cidrLast, cidrSize, formatIP, parseCidr } from '@/utils/ip'

export const MAX_COUNT = 1000

export type RandomBytes = (n: number) => Uint8Array

const cryptoBytes: RandomBytes = n => crypto.getRandomValues(new Uint8Array(n))

/** Uniform BigInt in [0, n) by rejection sampling over the smallest covering bit width. */
export function randomBelow(n: bigint, bytes: RandomBytes = cryptoBytes): bigint {
  if (n <= 1n)
    return 0n
  const bits = (n - 1n).toString(2).length
  const len = Math.ceil(bits / 8)
  const mask = (1n << BigInt(bits)) - 1n
  for (;;) {
    let v = 0n
    for (const b of bytes(len))
      v = (v << 8n) | BigInt(b)
    v &= mask
    if (v < n)
      return v
  }
}

export interface Options { count: number, unique: boolean, usableOnly: boolean }

export type GenerateError = 'network' | 'too-small'

/**
 * Random addresses inside the prefix. usableOnly skips the IPv4 network and broadcast addresses
 * (prefixes up to /30) and the IPv6 subnet-router anycast address (the all-zero host).
 */
export function generate(prefix: string, o: Options, bytes: RandomBytes = cryptoBytes): string[] | GenerateError {
  const net = parseCidr(prefix)
  if (!net)
    return 'network'
  const count = Math.max(1, Math.min(MAX_COUNT, Math.floor(o.count)))
  let first = net.network
  let last = cidrLast(net)
  if (o.usableOnly) {
    if (net.version === 4 && net.prefix <= 30) {
      first += 1n
      last -= 1n
    }
    else if (net.version === 6 && net.prefix < bitLength(6)) {
      first += 1n
    }
  }
  const size = last - first + 1n
  if (o.unique && size < BigInt(count))
    return 'too-small'
  const seen = new Set<bigint>()
  const out: string[] = []
  // When unique values would exhaust most of a small range, shuffle-sample instead of retrying.
  if (o.unique && size <= 4096n && BigInt(count) * 2n > size) {
    const all = Array.from({ length: Number(size) }, (_, i) => first + BigInt(i))
    for (let i = 0; i < count; i++) {
      const j = i + Number(randomBelow(BigInt(all.length - i), bytes))
      ;[all[i], all[j]] = [all[j]!, all[i]!]
      out.push(formatIP(net.version, all[i]!))
    }
    return out
  }
  while (out.length < count) {
    const v = first + randomBelow(size, bytes)
    if (o.unique) {
      if (seen.has(v))
        continue
      seen.add(v)
    }
    out.push(formatIP(net.version, v))
  }
  return out
}

export function prefixSize(prefix: string) {
  const n = parseCidr(prefix)
  return n ? cidrSize(n) : undefined
}
