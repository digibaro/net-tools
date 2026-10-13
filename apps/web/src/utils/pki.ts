import type { PublicKey } from '@/tools/certificate-decoder/x509'
import { Pkcs10CertificateRequest, X509Certificate } from '@/tools/certificate-decoder/x509'

/** PEM handling and key comparison shared by the PKI tools (everything runs in the browser). */

export interface PemBlock { label: string, der: Uint8Array<ArrayBuffer> }

const PEM_BLOCK = /-----BEGIN ([A-Z0-9 ]+)-----([\s\S]*?)-----END \1-----/g

export function pemBlocks(input: string): PemBlock[] {
  const out: PemBlock[] = []
  for (const m of input.matchAll(PEM_BLOCK)) {
    const b64 = m[2]!.replace(/^[\w-]+:.*$/gm, '').replace(/\s+/g, '')
    if (!/^[A-Z0-9+/]*={0,2}$/i.test(b64))
      continue
    try {
      out.push({ label: m[1]!, der: Uint8Array.from(atob(b64), c => c.charCodeAt(0)) })
    }
    catch {}
  }
  return out
}

export function toPem(label: string, der: ArrayBuffer | Uint8Array): string {
  const bytes = der instanceof Uint8Array ? der : new Uint8Array(der)
  let bin = ''
  for (const b of bytes)
    bin += String.fromCharCode(b)
  const b64 = btoa(bin).match(/.{1,64}/g)?.join('\n') ?? ''
  return `-----BEGIN ${label}-----\n${b64}\n-----END ${label}-----\n`
}

// --- minimal DER (enough to wrap PKCS#1 / SEC1 keys into PKCS#8) ---

interface Tlv { tag: number, start: number, end: number, next: number }

function readTlv(b: Uint8Array, off: number): Tlv {
  const tag = b[off]!
  let len = b[off + 1]!
  let p = off + 2
  if (len & 0x80) {
    const n = len & 0x7F
    if (n === 0 || n > 4)
      throw new Error('unsupported DER length')
    len = 0
    for (let i = 0; i < n; i++)
      len = len * 256 + b[p + i]!
    p += n
  }
  if (p + len > b.length)
    throw new Error('truncated DER')
  return { tag, start: p, end: p + len, next: p + len }
}

function children(b: Uint8Array, parent: Tlv): Tlv[] {
  const out: Tlv[] = []
  for (let off = parent.start; off < parent.end;) {
    const t = readTlv(b, off)
    out.push(t)
    off = t.next
  }
  return out
}

function encLen(n: number): number[] {
  if (n < 0x80)
    return [n]
  const bytes: number[] = []
  for (let v = n; v > 0; v = Math.floor(v / 256))
    bytes.unshift(v & 0xFF)
  return [0x80 | bytes.length, ...bytes]
}

function tlv(tag: number, ...parts: (Uint8Array | number[])[]): Uint8Array<ArrayBuffer> {
  const body = parts.flatMap(p => [...p])
  return Uint8Array.from([tag, ...encLen(body.length), ...body])
}

const OID_RSA = [0x06, 0x09, 0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x01]
const OID_EC = [0x06, 0x07, 0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x02, 0x01]

/** RSAPrivateKey (PKCS#1) → PrivateKeyInfo (PKCS#8). */
export function pkcs1ToPkcs8(pkcs1: Uint8Array): Uint8Array<ArrayBuffer> {
  return tlv(0x30, [0x02, 0x01, 0x00], tlv(0x30, OID_RSA, [0x05, 0x00]), tlv(0x04, pkcs1))
}

/** ECPrivateKey (SEC1, with the curve in [0]) → PrivateKeyInfo (PKCS#8). */
export function sec1ToPkcs8(sec1: Uint8Array): Uint8Array<ArrayBuffer> {
  const seq = readTlv(sec1, 0)
  const params = children(sec1, seq).find(c => c.tag === 0xA0)
  if (!params)
    throw new Error('SEC1 key without curve parameters')
  const curve = sec1.subarray(params.start, params.end) // the OID TLV inside [0]
  return tlv(0x30, [0x02, 0x01, 0x00], tlv(0x30, OID_EC, curve), tlv(0x04, sec1))
}

// --- keys ---

export type KeyError = 'no-key' | 'encrypted' | 'unsupported'

const IMPORT_CANDIDATES: (RsaHashedImportParams | EcKeyImportParams | Algorithm)[] = [
  { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-256' },
  { name: 'ECDSA', namedCurve: 'P-256' },
  { name: 'ECDSA', namedCurve: 'P-384' },
  { name: 'ECDSA', namedCurve: 'P-521' },
  { name: 'Ed25519' },
]

/** Imports a private key PEM (PKCS#8, PKCS#1 RSA or SEC1 EC) and returns its public key as SPKI DER. */
export async function privateKeySpki(input: string): Promise<{ spki: ArrayBuffer, description: string } | KeyError> {
  const blocks = pemBlocks(input)
  if (blocks.some(b => b.label === 'ENCRYPTED PRIVATE KEY') || /Proc-Type:.*ENCRYPTED/.test(input))
    return 'encrypted'
  const block = blocks.find(b => b.label.endsWith('PRIVATE KEY'))
  if (!block)
    return 'no-key'
  let pkcs8: Uint8Array<ArrayBuffer>
  try {
    pkcs8 = block.label === 'RSA PRIVATE KEY' ? pkcs1ToPkcs8(block.der) : block.label === 'EC PRIVATE KEY' ? sec1ToPkcs8(block.der) : block.der
  }
  catch {
    return 'unsupported'
  }
  for (const alg of IMPORT_CANDIDATES) {
    try {
      const priv = await crypto.subtle.importKey('pkcs8', pkcs8, alg, true, ['sign'])
      // The private JWK carries the public parameters (n/e or x/y); re-import them as a public key.
      const { d: _d, p: _p, q: _q, dp: _dp, dq: _dq, qi: _qi, ...pub } = await crypto.subtle.exportKey('jwk', priv)
      const pubKey = await crypto.subtle.importKey('jwk', { ...pub, key_ops: ['verify'] }, alg, true, ['verify'])
      return { spki: await crypto.subtle.exportKey('spki', pubKey), description: describeKey(priv.algorithm) }
    }
    catch {}
  }
  return 'unsupported'
}

export function describeKey(alg: KeyAlgorithm): string {
  const a = alg as KeyAlgorithm & { modulusLength?: number, namedCurve?: string }
  if (a.modulusLength)
    return `RSA ${a.modulusLength} bit`
  if (a.namedCurve)
    return `EC ${a.namedCurve}`
  return a.name
}

export async function sha256Hex(data: BufferSource): Promise<string> {
  const h = await crypto.subtle.digest('SHA-256', data)
  return [...new Uint8Array(h)].map(b => b.toString(16).padStart(2, '0')).join(':').toUpperCase()
}

export async function publicKeyDescription(pk: PublicKey): Promise<string> {
  try {
    return describeKey((await pk.export()).algorithm)
  }
  catch {
    return pk.algorithm.name
  }
}

export function certificateFrom(input: string): X509Certificate | undefined {
  const block = pemBlocks(input).find(b => b.label === 'CERTIFICATE')
  try {
    return block ? new X509Certificate(block.der) : undefined
  }
  catch {
    return undefined
  }
}

export function csrFrom(input: string): Pkcs10CertificateRequest | undefined {
  const block = pemBlocks(input).find(b => b.label === 'CERTIFICATE REQUEST' || b.label === 'NEW CERTIFICATE REQUEST')
  try {
    return block ? new Pkcs10CertificateRequest(block.der) : undefined
  }
  catch {
    return undefined
  }
}
