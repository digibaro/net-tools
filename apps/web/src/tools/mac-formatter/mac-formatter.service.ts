import { macHex, parseMac } from '@/utils/mac'

export type MacFormat = 'colon' | 'hyphen' | 'cisco' | 'plain'
export const MAC_FORMATS: MacFormat[] = ['colon', 'hyphen', 'cisco', 'plain']

export function formatAs(mac: bigint, format: MacFormat, upper = false): string {
  const hex = macHex(mac)
  const s = format === 'colon'
    ? hex.match(/../g)!.join(':')
    : format === 'hyphen'
      ? hex.match(/../g)!.join('-')
      : format === 'cisco'
        ? hex.match(/.{4}/g)!.join('.')
        : hex
  return upper ? s.toUpperCase() : s
}

/** Prefix of 1–5 octets (any separator) the generated MACs must start with. */
export function parsePrefix(input: string): { value: bigint, bits: number } | undefined {
  const s = input.trim().toLowerCase().replace(/[:.-]/g, '')
  if (!s)
    return { value: 0n, bits: 0 }
  if (!/^(?:[0-9a-f]{2}){1,5}$/.test(s))
    return undefined
  return { value: BigInt(`0x${s}`), bits: s.length * 4 }
}

/**
 * Random MACs. Without a prefix they are locally administered unicast (x2, x6, xA, xE first
 * octet), which never collides with vendor-assigned addresses.
 */
export function generateMacs(count: number, prefix: { value: bigint, bits: number }, random: (n: number) => Uint8Array = n => crypto.getRandomValues(new Uint8Array(n))): bigint[] {
  const out: bigint[] = []
  for (let i = 0; i < count; i++) {
    let mac = [...random(6)].reduce((acc, b) => (acc << 8n) | BigInt(b), 0n)
    if (prefix.bits === 0) {
      mac = (mac & ~(0x03n << 40n)) | (0x02n << 40n)
    }
    else {
      const shift = BigInt(48 - prefix.bits)
      mac = (prefix.value << shift) | (mac & ((1n << shift) - 1n))
    }
    out.push(mac)
  }
  return out
}

export function formatList(input: string, format: MacFormat, upper: boolean): { output: string[], invalid: string[] } {
  const output: string[] = []
  const invalid: string[] = []
  for (const s of input.split(/[\s,;]+/).filter(Boolean)) {
    const m = parseMac(s)
    if (m === undefined)
      invalid.push(s)
    else
      output.push(formatAs(m, format, upper))
  }
  return { output, invalid }
}
