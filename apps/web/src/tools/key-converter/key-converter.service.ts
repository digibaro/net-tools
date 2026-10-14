import type { Tlv } from './der'
import { bits, children, concat, decodeOid, DerError, expect, int, nul, octets, oid, readTlv, seq, tlv, uint } from './der'

// ---------- model ----------

export type Curve = 'P-256' | 'P-384' | 'P-521'

export type Key
  = | { kty: 'RSA', n: Uint8Array, e: Uint8Array, priv?: { d: Uint8Array, p: Uint8Array, q: Uint8Array, dp: Uint8Array, dq: Uint8Array, qi: Uint8Array } }
    | { kty: 'EC', crv: Curve, x: Uint8Array, y: Uint8Array, d?: Uint8Array }
    | { kty: 'OKP', crv: 'Ed25519', x: Uint8Array, d?: Uint8Array }

export type ErrorKey = 'unrecognized' | 'encrypted' | 'unsupportedAlg' | 'unsupportedCurve' | 'rsaJwkNoPrimes' | 'invalid' | 'noDerivation'

export class KeyError extends Error {
  constructor(public readonly key: ErrorKey, detail = '') {
    super(detail || key)
  }
}

const OID_RSA = '1.2.840.113549.1.1.1'
const OID_EC = '1.2.840.10045.2.1'
const OID_ED25519 = '1.3.101.112'
export const CURVES: Record<Curve, { oid: string, size: number, ssh: string }> = {
  'P-256': { oid: '1.2.840.10045.3.1.7', size: 32, ssh: 'nistp256' },
  'P-384': { oid: '1.3.132.0.34', size: 48, ssh: 'nistp384' },
  'P-521': { oid: '1.3.132.0.35', size: 66, ssh: 'nistp521' },
}
const curveByOid = (o: string) => (Object.keys(CURVES) as Curve[]).find(c => CURVES[c].oid === o)
const curveBySsh = (s: string) => (Object.keys(CURVES) as Curve[]).find(c => CURVES[c].ssh === s)

// ---------- bytes ----------

export const b64 = (b: Uint8Array) => btoa(String.fromCharCode(...b))
export const b64url = (b: Uint8Array) => b64(b).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
export function unb64(s: string): Uint8Array {
  const clean = s.replace(/\s+/g, '').replace(/-/g, '+').replace(/_/g, '/')
  if (!/^[A-Z0-9+/]*={0,2}$/i.test(clean))
    throw new KeyError('invalid', 'not base64')
  const bin = atob(clean + '='.repeat((4 - (clean.length % 4)) % 4))
  return Uint8Array.from(bin, c => c.charCodeAt(0))
}
function strip(b: Uint8Array) {
  let i = 0
  while (i < b.length - 1 && b[i] === 0) i++
  return b.subarray(i)
}
const pad = (b: Uint8Array, n: number) => (b.length >= n ? b.subarray(b.length - n) : concat(new Uint8Array(n - b.length), b))
const toBig = (b: Uint8Array) => (b.length ? BigInt(`0x${Array.from(b, x => x.toString(16).padStart(2, '0')).join('')}`) : 0n)
function fromBig(n: bigint): Uint8Array {
  let h = n.toString(16)
  if (h.length % 2)
    h = `0${h}`
  return Uint8Array.from(h.match(/../g)!, x => Number.parseInt(x, 16))
}
const utf8 = new TextEncoder()
const fromUtf8 = new TextDecoder()

async function sha256(b: Uint8Array): Promise<Uint8Array> {
  return new Uint8Array(await crypto.subtle.digest('SHA-256', b as Uint8Array<ArrayBuffer>))
}

// ---------- DER structures ----------

function rsaPublicFromPkcs1(der: Uint8Array): Key {
  const c = children(expect(readTlv(der), 0x30))
  return { kty: 'RSA', n: uint(c[0]), e: uint(c[1]) }
}

function rsaPrivateFromPkcs1(der: Uint8Array): Key {
  const c = children(expect(readTlv(der), 0x30))
  if (c.length < 9)
    throw new KeyError('invalid', 'RSA private key is incomplete')
  return { kty: 'RSA', n: uint(c[1]), e: uint(c[2]), priv: { d: uint(c[3]), p: uint(c[4]), q: uint(c[5]), dp: uint(c[6]), dq: uint(c[7]), qi: uint(c[8]) } }
}

function point(crv: Curve, q: Uint8Array): { x: Uint8Array, y: Uint8Array } {
  const n = CURVES[crv].size
  if (q[0] !== 0x04 || q.length !== 1 + 2 * n)
    throw new KeyError('invalid', 'only uncompressed EC points are supported')
  return { x: q.slice(1, 1 + n), y: q.slice(1 + n) }
}

const bitValue = (t: Tlv | undefined) => expect(t, 0x03).subarray(1)

function ecFromSec1(der: Uint8Array, crvHint?: Curve): { crv: Curve, d: Uint8Array, x?: Uint8Array, y?: Uint8Array } {
  const c = children(expect(readTlv(der), 0x30))
  const d = expect(c[1], 0x04)
  let crv = crvHint
  let pub: Uint8Array | undefined
  for (const t of c.slice(2)) {
    if (t.tag === 0xA0)
      crv = curveByOid(decodeOid(expect(readTlv(t.value), 0x06))) ?? crv
    if (t.tag === 0xA1)
      pub = bitValue(readTlv(t.value))
  }
  if (!crv)
    throw new KeyError('unsupportedCurve')
  return { crv, d: pad(d, CURVES[crv].size), ...(pub ? point(crv, pub) : {}) }
}

function algorithm(t: Tlv | undefined): { oid: string, param?: string } {
  const c = children(expect(t, 0x30))
  return { oid: decodeOid(expect(c[0], 0x06)), param: c[1]?.tag === 0x06 ? decodeOid(c[1].value) : undefined }
}

function fromSpki(der: Uint8Array): Key {
  const c = children(expect(readTlv(der), 0x30))
  const alg = algorithm(c[0])
  const key = bitValue(c[1])
  switch (alg.oid) {
    case OID_RSA:
      return rsaPublicFromPkcs1(key)
    case OID_EC: {
      const crv = alg.param ? curveByOid(alg.param) : undefined
      if (!crv)
        throw new KeyError('unsupportedCurve')
      return { kty: 'EC', crv, ...point(crv, key) }
    }
    case OID_ED25519:
      return { kty: 'OKP', crv: 'Ed25519', x: key.slice() }
  }
  throw new KeyError('unsupportedAlg', alg.oid)
}

/** Partially parsed private key: EC / Ed25519 may lack the public part (derived later). */
type Partial
  = | Key
    | { kty: 'EC', crv: Curve, d: Uint8Array, x?: undefined, y?: undefined }
    | { kty: 'OKP', crv: 'Ed25519', d: Uint8Array, x?: undefined }

function fromPkcs8(der: Uint8Array): Partial {
  const c = children(expect(readTlv(der), 0x30))
  const alg = algorithm(c[1])
  const priv = expect(c[2], 0x04)
  switch (alg.oid) {
    case OID_RSA:
      return rsaPrivateFromPkcs1(priv)
    case OID_EC: {
      const ec = ecFromSec1(priv, alg.param ? curveByOid(alg.param) : undefined)
      return ec.x ? { kty: 'EC', crv: ec.crv, x: ec.x, y: ec.y!, d: ec.d } : { kty: 'EC', crv: ec.crv, d: ec.d }
    }
    case OID_ED25519: {
      const d = expect(readTlv(priv), 0x04)
      const pubTlv = c.find(t => t.tag === 0x81) // OneAsymmetricKey [1] publicKey
      return pubTlv ? { kty: 'OKP', crv: 'Ed25519', d: d.slice(), x: pubTlv.value.subarray(1).slice() } : { kty: 'OKP', crv: 'Ed25519', d: d.slice() }
    }
  }
  throw new KeyError('unsupportedAlg', alg.oid)
}

/** SubjectPublicKeyInfo of an X.509 certificate. */
function spkiOfCertificate(der: Uint8Array): Uint8Array {
  const cert = children(expect(readTlv(der), 0x30))
  const tbs = children(expect(cert[0], 0x30))
  const i = tbs[0]?.tag === 0xA0 ? 1 : 0
  const spki = tbs[i + 5]
  if (!spki || spki.tag !== 0x30)
    throw new KeyError('invalid', 'certificate without a public key')
  return seq(spki.value)
}

// ---------- SSH wire format ----------

class SshReader {
  private off = 0
  constructor(private readonly b: Uint8Array) {}
  u32(): number {
    if (this.off + 4 > this.b.length)
      throw new KeyError('invalid', 'truncated SSH data')
    const v = ((this.b[this.off]! << 24) | (this.b[this.off + 1]! << 16) | (this.b[this.off + 2]! << 8) | this.b[this.off + 3]!) >>> 0
    this.off += 4
    return v
  }

  bytes(): Uint8Array {
    const n = this.u32()
    if (this.off + n > this.b.length)
      throw new KeyError('invalid', 'truncated SSH data')
    const v = this.b.subarray(this.off, this.off + n)
    this.off += n
    return v
  }

  str(): string {
    return fromUtf8.decode(this.bytes())
  }

  mpint(): Uint8Array {
    return strip(this.bytes())
  }

  get done(): boolean {
    return this.off >= this.b.length
  }
}

const u32 = (n: number) => new Uint8Array([(n >>> 24) & 255, (n >>> 16) & 255, (n >>> 8) & 255, n & 255])
const sshBytes = (b: Uint8Array) => concat(u32(b.length), b)
const sshStr = (s: string) => sshBytes(utf8.encode(s))
function sshMpint(b: Uint8Array): Uint8Array {
  const s = strip(b)
  if (s.length === 1 && s[0] === 0)
    return u32(0)
  return sshBytes(s[0]! & 0x80 ? concat(new Uint8Array([0]), s) : s)
}

function sshType(k: Key): string {
  return k.kty === 'RSA' ? 'ssh-rsa' : k.kty === 'EC' ? `ecdsa-sha2-${CURVES[k.crv].ssh}` : 'ssh-ed25519'
}

/** The SSH public key blob (RFC 4253, 5656, 8709). */
export function sshBlob(k: Key): Uint8Array {
  switch (k.kty) {
    case 'RSA':
      return concat(sshStr('ssh-rsa'), sshMpint(k.e), sshMpint(k.n))
    case 'EC':
      return concat(sshStr(sshType(k)), sshStr(CURVES[k.crv].ssh), sshBytes(concat(new Uint8Array([4]), k.x, k.y)))
    case 'OKP':
      return concat(sshStr('ssh-ed25519'), sshBytes(k.x))
  }
}

function fromSshBlob(blob: Uint8Array): Key {
  const r = new SshReader(blob)
  const type = r.str()
  if (type === 'ssh-rsa') {
    const e = r.mpint()
    return { kty: 'RSA', e, n: r.mpint() }
  }
  if (type === 'ssh-ed25519')
    return { kty: 'OKP', crv: 'Ed25519', x: r.bytes().slice() }
  if (type.startsWith('ecdsa-sha2-')) {
    const crv = curveBySsh(r.str())
    if (!crv)
      throw new KeyError('unsupportedCurve')
    return { kty: 'EC', crv, ...point(crv, r.bytes()) }
  }
  throw new KeyError('unsupportedAlg', type)
}

const OPENSSH_MAGIC = 'openssh-key-v1\0'

function fromOpenSshPrivate(der: Uint8Array): { key: Key, comment: string } {
  if (fromUtf8.decode(der.subarray(0, OPENSSH_MAGIC.length)) !== OPENSSH_MAGIC)
    throw new KeyError('invalid', 'not an OpenSSH private key')
  const r = new SshReader(der.subarray(OPENSSH_MAGIC.length))
  const cipher = r.str()
  r.str() // kdf name
  r.bytes() // kdf options
  if (cipher !== 'none')
    throw new KeyError('encrypted')
  if (r.u32() !== 1)
    throw new KeyError('invalid', 'only single-key files are supported')
  r.bytes() // public blob (repeated in the private section)
  const p = new SshReader(r.bytes())
  const check1 = p.u32()
  const check2 = p.u32()
  if (check1 !== check2)
    throw new KeyError('invalid', 'check bytes do not match')
  const type = p.str()
  let key: Key
  if (type === 'ssh-rsa') {
    const n = p.mpint()
    const e = p.mpint()
    const d = p.mpint()
    const qi = p.mpint()
    const pp = p.mpint()
    const q = p.mpint()
    const dBig = toBig(d)
    key = { kty: 'RSA', n, e, priv: { d, p: pp, q, qi, dp: fromBig(dBig % (toBig(pp) - 1n)), dq: fromBig(dBig % (toBig(q) - 1n)) } }
  }
  else if (type === 'ssh-ed25519') {
    const x = p.bytes().slice()
    const both = p.bytes()
    key = { kty: 'OKP', crv: 'Ed25519', x, d: both.slice(0, 32) }
  }
  else if (type.startsWith('ecdsa-sha2-')) {
    const crv = curveBySsh(p.str())
    if (!crv)
      throw new KeyError('unsupportedCurve')
    const pt = point(crv, p.bytes())
    key = { kty: 'EC', crv, ...pt, d: pad(p.mpint(), CURVES[crv].size) }
  }
  else {
    throw new KeyError('unsupportedAlg', type)
  }
  return { key, comment: p.str() }
}

// ---------- JWK ----------

interface Jwk { kty?: string, crv?: string, n?: string, e?: string, d?: string, p?: string, q?: string, dp?: string, dq?: string, qi?: string, x?: string, y?: string, keys?: Jwk[] }

function fromJwk(j: Jwk): Key {
  const f = (v?: string) => {
    if (!v)
      throw new KeyError('invalid', 'JWK is missing a member')
    return unb64(v)
  }
  if (j.kty === 'RSA') {
    const k: Key = { kty: 'RSA', n: strip(f(j.n)), e: strip(f(j.e)) }
    if (j.d) {
      if (!j.p || !j.q || !j.dp || !j.dq || !j.qi)
        throw new KeyError('rsaJwkNoPrimes')
      k.priv = { d: strip(f(j.d)), p: strip(f(j.p)), q: strip(f(j.q)), dp: strip(f(j.dp)), dq: strip(f(j.dq)), qi: strip(f(j.qi)) }
    }
    return k
  }
  if (j.kty === 'EC') {
    const crv = j.crv as Curve
    if (!CURVES[crv])
      throw new KeyError('unsupportedCurve', j.crv)
    return { kty: 'EC', crv, x: f(j.x), y: f(j.y), ...(j.d ? { d: f(j.d) } : {}) }
  }
  if (j.kty === 'OKP') {
    if (j.crv !== 'Ed25519')
      throw new KeyError('unsupportedCurve', j.crv)
    return { kty: 'OKP', crv: 'Ed25519', x: f(j.x), ...(j.d ? { d: f(j.d) } : {}) }
  }
  throw new KeyError('unsupportedAlg', j.kty)
}

export function toJwk(k: Key, withPrivate: boolean): Record<string, string> {
  switch (k.kty) {
    case 'RSA':
      return { kty: 'RSA', n: b64url(k.n), e: b64url(k.e), ...(withPrivate && k.priv ? { d: b64url(k.priv.d), p: b64url(k.priv.p), q: b64url(k.priv.q), dp: b64url(k.priv.dp), dq: b64url(k.priv.dq), qi: b64url(k.priv.qi) } : {}) }
    case 'EC':
      return { kty: 'EC', crv: k.crv, x: b64url(k.x), y: b64url(k.y), ...(withPrivate && k.d ? { d: b64url(k.d) } : {}) }
    case 'OKP':
      return { kty: 'OKP', crv: 'Ed25519', x: b64url(k.x), ...(withPrivate && k.d ? { d: b64url(k.d) } : {}) }
  }
}

// ---------- writers ----------

export const pkcs1Public = (k: Extract<Key, { kty: 'RSA' }>) => seq(int(k.n), int(k.e))

export function spki(k: Key): Uint8Array {
  switch (k.kty) {
    case 'RSA':
      return seq(seq(oid(OID_RSA), nul()), bits(pkcs1Public(k)))
    case 'EC':
      return seq(seq(oid(OID_EC), oid(CURVES[k.crv].oid)), bits(concat(new Uint8Array([4]), k.x, k.y)))
    case 'OKP':
      return seq(seq(oid(OID_ED25519)), bits(k.x))
  }
}

export function pkcs1Private(k: Extract<Key, { kty: 'RSA' }>): Uint8Array {
  const p = k.priv!
  return seq(int(0), int(k.n), int(k.e), int(p.d), int(p.p), int(p.q), int(p.dp), int(p.dq), int(p.qi))
}

/** SEC1 ECPrivateKey; with params for the standalone "EC PRIVATE KEY" form. */
export function sec1(k: Extract<Key, { kty: 'EC' }>, withParams: boolean): Uint8Array {
  return seq(int(1), octets(pad(k.d!, CURVES[k.crv].size)), ...(withParams ? [tlv(0xA0, oid(CURVES[k.crv].oid))] : []), tlv(0xA1, bits(concat(new Uint8Array([4]), k.x, k.y))))
}

export function pkcs8(k: Key): Uint8Array {
  switch (k.kty) {
    case 'RSA':
      return seq(int(0), seq(oid(OID_RSA), nul()), octets(pkcs1Private(k)))
    case 'EC':
      return seq(int(0), seq(oid(OID_EC), oid(CURVES[k.crv].oid)), octets(sec1(k, false)))
    case 'OKP':
      return seq(int(0), seq(oid(OID_ED25519)), octets(octets(k.d!)))
  }
}

export function pem(label: string, der: Uint8Array, width = 64): string {
  const body = b64(der).match(new RegExp(`.{1,${width}}`, 'g'))!.join('\n')
  return `-----BEGIN ${label}-----\n${body}\n-----END ${label}-----`
}

export const sshPublic = (k: Key, comment = '') => `${sshType(k)} ${b64(sshBlob(k))}${comment ? ` ${comment}` : ''}`

/** Unencrypted OpenSSH private key ("openssh-key-v1"). checkint is fixed so output is stable. */
export function opensshPrivate(k: Key, comment = '', checkint = 0x4E455454): string {
  let body: Uint8Array
  switch (k.kty) {
    case 'RSA':
      body = concat(sshStr('ssh-rsa'), sshMpint(k.n), sshMpint(k.e), sshMpint(k.priv!.d), sshMpint(k.priv!.qi), sshMpint(k.priv!.p), sshMpint(k.priv!.q))
      break
    case 'EC':
      body = concat(sshStr(sshType(k)), sshStr(CURVES[k.crv].ssh), sshBytes(concat(new Uint8Array([4]), k.x, k.y)), sshMpint(k.d!))
      break
    case 'OKP':
      body = concat(sshStr('ssh-ed25519'), sshBytes(k.x), sshBytes(concat(k.d!, k.x)))
      break
  }
  let priv = concat(u32(checkint), u32(checkint), body, sshStr(comment))
  const padLen = (8 - (priv.length % 8)) % 8
  priv = concat(priv, Uint8Array.from({ length: padLen }, (_, i) => i + 1))
  const der = concat(utf8.encode(OPENSSH_MAGIC), sshStr('none'), sshStr('none'), sshBytes(new Uint8Array()), u32(1), sshBytes(sshBlob(k)), sshBytes(priv))
  return pem('OPENSSH PRIVATE KEY', der, 70)
}

// ---------- derivation of missing public parts (WebCrypto) ----------

async function complete(p: Partial): Promise<Key> {
  if (p.kty === 'RSA' || p.x)
    return p as Key
  const der = p.kty === 'EC'
    ? seq(int(0), seq(oid(OID_EC), oid(CURVES[p.crv].oid)), octets(seq(int(1), octets(p.d))))
    : seq(int(0), seq(oid(OID_ED25519)), octets(octets(p.d)))
  try {
    const alg = p.kty === 'EC' ? { name: 'ECDSA', namedCurve: p.crv } : { name: 'Ed25519' }
    const ck = await crypto.subtle.importKey('pkcs8', der as Uint8Array<ArrayBuffer>, alg, true, ['sign'])
    const jwk = await crypto.subtle.exportKey('jwk', ck)
    return fromJwk(jwk as Jwk)
  }
  catch {
    throw new KeyError('noDerivation')
  }
}

// ---------- input detection ----------

export interface Parsed {
  key: Key
  source: string
  comment: string
}

const PEM_RE = /-----BEGIN ([A-Z0-9 ]+)-----([\s\S]*?)-----END \1-----/g

export async function parseKey(input: string): Promise<Parsed> {
  const s = input.trim()
  if (!s)
    throw new KeyError('unrecognized')
  try {
    if (s.startsWith('{')) {
      let j: Jwk
      try {
        j = JSON.parse(s) as Jwk
      }
      catch {
        throw new KeyError('invalid', 'invalid JSON')
      }
      const one = j.keys?.[0] ?? j
      return { key: fromJwk(one), source: j.keys ? 'JWKS' : 'JWK', comment: '' }
    }
    const words = s.split(/\s+/)
    if (/^(?:ssh-rsa|ssh-ed25519|ecdsa-sha2-nistp\d+)$/.test(words[0]!) && words[1])
      return { key: fromSshBlob(unb64(words[1])), source: 'OpenSSH public key', comment: words.slice(2).join(' ') }
    for (const m of s.matchAll(PEM_RE)) {
      const label = m[1]!
      const body = m[2]!
      if (/Proc-Type:\s*4,ENCRYPTED/i.test(body) || label === 'ENCRYPTED PRIVATE KEY')
        throw new KeyError('encrypted')
      if (label === 'EC PARAMETERS')
        continue
      const der = unb64(body.replace(/^[\w-]+:.*$/gm, ''))
      switch (label) {
        case 'PUBLIC KEY': return { key: fromSpki(der), source: 'PEM SubjectPublicKeyInfo', comment: '' }
        case 'RSA PUBLIC KEY': return { key: rsaPublicFromPkcs1(der), source: 'PEM PKCS#1 public', comment: '' }
        case 'PRIVATE KEY': return { key: await complete(fromPkcs8(der)), source: 'PEM PKCS#8 private', comment: '' }
        case 'RSA PRIVATE KEY': return { key: rsaPrivateFromPkcs1(der), source: 'PEM PKCS#1 private', comment: '' }
        case 'EC PRIVATE KEY': {
          const ec = ecFromSec1(der)
          return { key: await complete(ec.x ? { kty: 'EC', crv: ec.crv, x: ec.x, y: ec.y!, d: ec.d } : { kty: 'EC', crv: ec.crv, d: ec.d }), source: 'PEM SEC1 EC private', comment: '' }
        }
        case 'OPENSSH PRIVATE KEY': {
          const o = fromOpenSshPrivate(der)
          return { key: o.key, source: 'OpenSSH private key', comment: o.comment }
        }
        case 'CERTIFICATE': return { key: fromSpki(spkiOfCertificate(der)), source: 'X.509 certificate (public key)', comment: '' }
        default: throw new KeyError('unrecognized', label)
      }
    }
    if (s.length >= 40 && /^[A-Z0-9+/=\s]+$/i.test(s)) { // bare base64 DER: try public, then private
      const der = unb64(s)
      try {
        return { key: fromSpki(der), source: 'DER SubjectPublicKeyInfo', comment: '' }
      }
      catch {
        return { key: await complete(fromPkcs8(der)), source: 'DER PKCS#8 private', comment: '' }
      }
    }
  }
  catch (e) {
    if (e instanceof KeyError)
      throw e
    if (e instanceof DerError)
      throw new KeyError('invalid', e.message)
    throw new KeyError('invalid')
  }
  throw new KeyError('unrecognized')
}

// ---------- outputs ----------

export interface Output {
  id: string
  text: string
  private: boolean
}

export interface Converted {
  type: string
  bits: number
  isPrivate: boolean
  outputs: Output[]
  fingerprints: { sshSha256: string, jwkThumbprint: string, spkiSha256: string }
}

export function keyBits(k: Key): number {
  if (k.kty === 'RSA')
    return strip(k.n).length * 8 - Math.clz32(strip(k.n)[0]!) + 24
  return k.kty === 'EC' ? Number(k.crv.slice(2)) : 256
}

/** RFC 7638 JWK thumbprint (SHA-256, base64url). */
export async function jwkThumbprint(k: Key): Promise<string> {
  const j = toJwk(k, false)
  const members = k.kty === 'RSA' ? ['e', 'kty', 'n'] : k.kty === 'EC' ? ['crv', 'kty', 'x', 'y'] : ['crv', 'kty', 'x']
  const canonical = `{${members.map(m => `"${m}":"${j[m]}"`).join(',')}}`
  return b64url(await sha256(utf8.encode(canonical)))
}

export async function convert(p: Parsed): Promise<Converted> {
  const k = p.key
  const isPrivate = k.kty === 'RSA' ? !!k.priv : !!k.d
  const outputs: Output[] = [
    { id: 'jwkPublic', text: JSON.stringify(toJwk(k, false), null, 2), private: false },
    { id: 'pemSpki', text: pem('PUBLIC KEY', spki(k)), private: false },
    { id: 'sshPublic', text: sshPublic(k, p.comment), private: false },
  ]
  if (k.kty === 'RSA')
    outputs.push({ id: 'pemPkcs1Public', text: pem('RSA PUBLIC KEY', pkcs1Public(k)), private: false })
  if (isPrivate) {
    outputs.push({ id: 'jwkPrivate', text: JSON.stringify(toJwk(k, true), null, 2), private: true })
    outputs.push({ id: 'pemPkcs8', text: pem('PRIVATE KEY', pkcs8(k)), private: true })
    if (k.kty === 'RSA')
      outputs.push({ id: 'pemPkcs1Private', text: pem('RSA PRIVATE KEY', pkcs1Private(k)), private: true })
    if (k.kty === 'EC')
      outputs.push({ id: 'pemSec1', text: pem('EC PRIVATE KEY', sec1(k, true)), private: true })
    outputs.push({ id: 'opensshPrivate', text: opensshPrivate(k, p.comment), private: true })
  }
  return {
    type: k.kty === 'RSA' ? 'RSA' : k.kty === 'EC' ? `ECDSA ${k.crv}` : 'Ed25519',
    bits: keyBits(k),
    isPrivate,
    outputs,
    fingerprints: {
      sshSha256: `SHA256:${b64(await sha256(sshBlob(k))).replace(/=+$/, '')}`,
      jwkThumbprint: await jwkThumbprint(k),
      spkiSha256: b64(await sha256(spki(k))),
    },
  }
}
