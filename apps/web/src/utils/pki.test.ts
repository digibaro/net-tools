import { generateKeyPairSync } from 'node:crypto'
import { describe, expect, it } from 'vitest'
import { pemBlocks, privateKeySpki, sha256Hex, toPem } from './pki'

describe('pem', () => {
  it('round-trips', () => {
    const pem = toPem('TEST', Uint8Array.from([1, 2, 3]))
    expect(pemBlocks(`junk\n${pem}junk`)).toEqual([{ label: 'TEST', der: Uint8Array.from([1, 2, 3]) }])
  })
})

describe('privateKeySpki', () => {
  it.each([
    ['rsa', 'pkcs8'],
    ['rsa', 'pkcs1'],
    ['ec', 'pkcs8'],
    ['ec', 'sec1'],
    ['ed25519', 'pkcs8'],
  ] as const)('%s %s', async (type, format) => {
    const opts = type === 'rsa' ? { modulusLength: 2048 } : type === 'ec' ? { namedCurve: 'P-384' } : {}
    const { publicKey, privateKey } = generateKeyPairSync(type as 'rsa', opts as { modulusLength: number })
    const pem = privateKey.export({ type: format as 'pkcs8', format: 'pem' }) as string
    const r = await privateKeySpki(pem)
    if (typeof r === 'string')
      throw new Error(r)
    const want = publicKey.export({ type: 'spki', format: 'der' })
    expect(await sha256Hex(r.spki)).toBe(await sha256Hex(new Uint8Array(want)))
    expect(r.description).toBe(type === 'rsa' ? 'RSA 2048 bit' : type === 'ec' ? 'EC P-384' : 'Ed25519')
  })
  it('reports encrypted and missing keys', async () => {
    const { privateKey } = generateKeyPairSync('ec', { namedCurve: 'P-256' })
    const enc = privateKey.export({ type: 'pkcs8', format: 'pem', cipher: 'aes-256-cbc', passphrase: 'x' }) as string
    expect(await privateKeySpki(enc)).toBe('encrypted')
    expect(await privateKeySpki('nothing')).toBe('no-key')
  })
})
