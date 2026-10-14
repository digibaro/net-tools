import type { Key } from './key-converter.service'
import { describe, expect, it } from 'vitest'
import { SSH_FIXTURES } from './key-converter.fixtures'
import { b64, convert, opensshPrivate, parseKey, pem, pkcs8, sec1, spki, toJwk } from './key-converter.service'

const subtle = globalThis.crypto.subtle

async function generate(alg: RsaHashedKeyGenParams | EcKeyGenParams | { name: 'Ed25519' }) {
  const pair = await subtle.generateKey(alg, true, ['sign', 'verify']) as CryptoKeyPair
  return {
    pkcs8: new Uint8Array(await subtle.exportKey('pkcs8', pair.privateKey)),
    spki: new Uint8Array(await subtle.exportKey('spki', pair.publicKey)),
    jwk: await subtle.exportKey('jwk', pair.privateKey),
  }
}

const ALGS = [
  { name: 'RSASSA-PKCS1-v1_5', modulusLength: 2048, publicExponent: new Uint8Array([1, 0, 1]), hash: 'SHA-256' },
  { name: 'ECDSA', namedCurve: 'P-256' },
  { name: 'ECDSA', namedCurve: 'P-384' },
  { name: 'ECDSA', namedCurve: 'P-521' },
  { name: 'Ed25519' },
] as const

describe('key-converter against WebCrypto', () => {
  it.each(ALGS.map(a => ['namedCurve' in a ? a.namedCurve : a.name, a]))('%s: PKCS#8, SPKI and JWK round-trip', async (_, alg) => {
    const g = await generate(alg as never)
    const { key } = await parseKey(pem('PRIVATE KEY', g.pkcs8))
    expect(b64(spki(key))).toBe(b64(g.spki))
    expect(toJwk(key, true)).toEqual(Object.fromEntries(Object.entries(g.jwk).filter(([k]) => ['kty', 'crv', 'n', 'e', 'd', 'p', 'q', 'dp', 'dq', 'qi', 'x', 'y'].includes(k))))
    // Our PKCS#8 must import back into WebCrypto and match the original JWK.
    const back = await subtle.importKey('pkcs8', pkcs8(key) as Uint8Array<ArrayBuffer>, alg as never, true, ['sign'])
    expect((await subtle.exportKey('jwk', back)).d).toBe(g.jwk.d)
    // JWK input, OpenSSH private round trip and SPKI-only input agree.
    const fromJwk = (await parseKey(JSON.stringify(g.jwk))).key
    expect(b64(spki(fromJwk))).toBe(b64(g.spki))
    const fromSsh = (await parseKey(opensshPrivate(key, 'c'))).key
    expect(toJwk(fromSsh, true)).toEqual(toJwk(key, true))
    expect(b64(spki((await parseKey(pem('PUBLIC KEY', g.spki))).key))).toBe(b64(g.spki))
  }, 30000)

  it('derives the public point of an EC key without one', async () => {
    const g = await generate({ name: 'ECDSA', namedCurve: 'P-256' })
    const { key } = await parseKey(pem('PRIVATE KEY', g.pkcs8))
    const k = key as Extract<Key, { kty: 'EC' }>
    // SEC1 without the optional public key: the converter must recompute x and y.
    const bare = sec1(k, true)
    const stripped = bare.slice(0, bare.length - (2 + 2 + 1 + 65)) // drop the [1] publicKey element
    stripped[1] = stripped.length - 2
    const parsed = (await parseKey(pem('EC PRIVATE KEY', stripped))).key as Extract<Key, { kty: 'EC' }>
    expect(b64(parsed.x)).toBe(b64(k.x))
    expect(b64(parsed.y)).toBe(b64(k.y))
  })
})

describe('key-converter against ssh-keygen', () => {
  it.each(Object.entries(SSH_FIXTURES))('%s public key, fingerprint and PKCS8 export', async (_, f) => {
    const p = await parseKey(f.pub)
    expect(p.comment).toBe('test@nettools')
    const c = await convert(p)
    expect(c.fingerprints.sshSha256).toBe(f.fingerprint)
    expect(c.outputs.find(o => o.id === 'sshPublic')!.text).toBe(f.pub)
    if (f.spki)
      expect(c.outputs.find(o => o.id === 'pemSpki')!.text).toBe(f.spki)
    expect(c.isPrivate).toBe(false)
  })
})

describe('key-converter errors and extras', () => {
  const err = async (s: string) => parseKey(s).then(() => 'ok', (e: { key: string }) => e.key)
  it('refuses encrypted and unknown input', async () => {
    expect(await err('-----BEGIN ENCRYPTED PRIVATE KEY-----\nAAAA\n-----END ENCRYPTED PRIVATE KEY-----')).toBe('encrypted')
    expect(await err('-----BEGIN RSA PRIVATE KEY-----\nProc-Type: 4,ENCRYPTED\nDEK-Info: AES-128-CBC,00\n\nAAAA\n-----END RSA PRIVATE KEY-----')).toBe('encrypted')
    expect(await err('hello')).toBe('unrecognized')
    expect(await err('{"kty":"RSA","n":"AQAB","e":"AQAB","d":"AQAB"}')).toBe('rsaJwkNoPrimes')
    expect(await err('{"kty":"EC","crv":"secp256k1","x":"AA","y":"AA"}')).toBe('unsupportedCurve')
    expect(await err('-----BEGIN PUBLIC KEY-----\nMAA=\n-----END PUBLIC KEY-----')).toBe('invalid')
  })
  it('computes the RFC 7638 example thumbprint', async () => {
    const jwk = { kty: 'RSA', e: 'AQAB', n: '0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw' }
    const c = await convert(await parseKey(JSON.stringify(jwk)))
    expect(c.fingerprints.jwkThumbprint).toBe('NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs')
    expect(c.bits).toBe(2048)
    expect(c.type).toBe('RSA')
  })
})
