import { formatIPv4, formatIPv6 } from '@/utils/ip'

export type InputForm = 'dotted' | 'binary' | 'inet_aton'

function parsePart(p: string): bigint | undefined {
  if (/^0x[0-9a-f]+$/i.test(p))
    return BigInt(p)
  if (/^0[0-7]*$/.test(p))
    return BigInt(`0o${p.slice(1) || '0'}`)
  if (/^[1-9]\d*$/.test(p))
    return BigInt(p)
  return undefined
}

/**
 * Parses an IPv4 address in any common notation: dotted binary, 32-bit binary, or the forms
 * inet_aton(3) accepts (1–4 parts, each decimal, 0x-hex or 0-octal; the last part fills the
 * remaining bytes), e.g. 127.1, 0177.0.0.1, 0x7f.1, 2130706433, 0x7f000001.
 */
export function parseAnyIPv4(input: string): { value: bigint, form: InputForm } | undefined {
  const s = input.trim().toLowerCase()
  if (/^[01]{8}(?:\.[01]{8}){3}$/.test(s))
    return { value: BigInt(`0b${s.replaceAll('.', '')}`), form: 'binary' }
  if (/^0b[01]{1,32}$/.test(s))
    return { value: BigInt(s), form: 'binary' }
  const parts = s.split('.')
  if (parts.length < 1 || parts.length > 4)
    return undefined
  const nums = parts.map(parsePart)
  if (nums.includes(undefined))
    return undefined
  const values = nums as bigint[]
  const lastBits = BigInt(8 * (5 - values.length))
  let value = 0n
  for (let i = 0; i < values.length - 1; i++) {
    if (values[i]! > 255n)
      return undefined
    value = (value << 8n) | values[i]!
  }
  const last = values.at(-1)!
  if (last >= 1n << lastBits)
    return undefined
  value = (value << lastBits) | last
  const dotted = /^(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/.test(s)
  return { value, form: dotted ? 'dotted' : 'inet_aton' }
}

export interface Ipv4Formats {
  dotted: string
  decimal: string
  hex: string
  hexDotted: string
  octalDotted: string
  binaryDotted: string
  binary: string
  ipv4Mapped: string
}

const bytes = (v: bigint) => [24n, 16n, 8n, 0n].map(s => (v >> s) & 0xFFn)

export function formats(value: bigint): Ipv4Formats {
  return {
    dotted: formatIPv4(value),
    decimal: value.toString(),
    hex: `0x${value.toString(16).toUpperCase().padStart(8, '0')}`,
    hexDotted: bytes(value).map(b => `0x${b.toString(16).toUpperCase().padStart(2, '0')}`).join('.'),
    octalDotted: bytes(value).map(b => `0${b.toString(8).padStart(3, '0')}`).join('.'),
    binaryDotted: bytes(value).map(b => b.toString(2).padStart(8, '0')).join('.'),
    binary: value.toString(2).padStart(32, '0'),
    ipv4Mapped: `::ffff:${formatIPv4(value)} (${formatIPv6((0xFFFFn << 32n) | value)})`,
  }
}
