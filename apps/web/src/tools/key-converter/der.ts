/** Minimal DER codec for the key structures the converter needs (no BER, no indefinite lengths). */

export interface Tlv {
  tag: number
  value: Uint8Array
  /** Offset just past this element in the parent buffer. */
  end: number
}

export class DerError extends Error {}

export function readTlv(b: Uint8Array, off = 0): Tlv {
  if (off + 2 > b.length)
    throw new DerError('truncated DER')
  const tag = b[off]!
  let len = b[off + 1]!
  let p = off + 2
  if (len & 0x80) {
    const n = len & 0x7F
    if (n === 0 || n > 4 || p + n > b.length)
      throw new DerError('bad DER length')
    len = 0
    for (let i = 0; i < n; i++) len = len * 256 + b[p + i]!
    p += n
  }
  if (p + len > b.length)
    throw new DerError('truncated DER')
  return { tag, value: b.subarray(p, p + len), end: p + len }
}

/** Children of a constructed value (SEQUENCE, context tags). */
export function children(value: Uint8Array): Tlv[] {
  const out: Tlv[] = []
  for (let off = 0; off < value.length;) {
    const t = readTlv(value, off)
    out.push(t)
    off = t.end
  }
  return out
}

export function expect(t: Tlv | undefined, tag: number): Uint8Array {
  if (!t || t.tag !== tag)
    throw new DerError(`expected tag 0x${tag.toString(16)}`)
  return t.value
}

/** Unsigned big-endian integer bytes without sign padding. */
export function uint(t: Tlv | undefined): Uint8Array {
  const v = expect(t, 0x02)
  let i = 0
  while (i < v.length - 1 && v[i] === 0) i++
  return v.subarray(i)
}

export function decodeOid(v: Uint8Array): string {
  const parts = [Math.floor(v[0]! / 40), v[0]! % 40]
  let n = 0
  for (let i = 1; i < v.length; i++) {
    n = n * 128 + (v[i]! & 0x7F)
    if (!(v[i]! & 0x80)) {
      parts.push(n)
      n = 0
    }
  }
  return parts.join('.')
}

// ---------- encoding ----------

function lenBytes(n: number): number[] {
  if (n < 0x80)
    return [n]
  const out: number[] = []
  for (; n > 0; n = Math.floor(n / 256)) out.unshift(n & 0xFF)
  return [0x80 | out.length, ...out]
}

export function concat(...parts: Uint8Array[]): Uint8Array {
  const out = new Uint8Array(parts.reduce((n, p) => n + p.length, 0))
  let off = 0
  for (const p of parts) {
    out.set(p, off)
    off += p.length
  }
  return out
}

export function tlv(tag: number, ...content: Uint8Array[]) {
  const c = concat(...content)
  return concat(new Uint8Array([tag, ...lenBytes(c.length)]), c)
}

export const seq = (...parts: Uint8Array[]) => tlv(0x30, ...parts)
export const octets = (v: Uint8Array) => tlv(0x04, v)
export const bits = (v: Uint8Array) => tlv(0x03, new Uint8Array([0]), v)
export const nul = () => new Uint8Array([0x05, 0x00])

/** INTEGER from unsigned big-endian bytes (adds a 0x00 when the high bit is set). */
export function int(v: Uint8Array | number): Uint8Array {
  let b = typeof v === 'number' ? new Uint8Array([v]) : v
  let i = 0
  while (i < b.length - 1 && b[i] === 0) i++
  b = b.subarray(i)
  return tlv(0x02, b[0]! & 0x80 ? concat(new Uint8Array([0]), b) : b)
}

export function oid(s: string): Uint8Array {
  const p = s.split('.').map(Number)
  const out = [p[0]! * 40 + p[1]!]
  for (const n of p.slice(2)) {
    const enc = [n & 0x7F]
    for (let x = Math.floor(n / 128); x > 0; x = Math.floor(x / 128)) enc.unshift((x & 0x7F) | 0x80)
    out.push(...enc)
  }
  return tlv(0x06, new Uint8Array(out))
}
