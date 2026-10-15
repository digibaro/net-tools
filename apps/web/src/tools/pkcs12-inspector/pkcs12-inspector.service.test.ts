import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { readPkcs12 } from './pkcs12-inspector.service'

const fixture = (name: string) => new Uint8Array(readFileSync(join(__dirname, 'testdata', name)))

describe('pkcs12-inspector', () => {
  it('reads a modern (OpenSSL 3) EC bundle', async () => {
    const r = await readPkcs12(fixture('ec.p12'), 'secret')
    if (typeof r === 'string')
      throw new Error(r)
    expect(r.certs).toHaveLength(1)
    expect(r.certs[0]!.subject).toBe('CN=ec.example.test')
    expect(r.certs[0]!.friendlyName).toBe('ec test')
    expect(r.keys[0]!.matchesCert).toBe(0)
    expect(r.keys[0]!.description).toMatch(/P-256/)
    expect(r.mac).toBe('SHA-256, 2048 iterations')
    expect(r.encryption.join(' ')).toContain('AES-256-CBC')
    expect(r.legacy).toBe(false)
  })
  it('reads a legacy RSA bundle and flags it', async () => {
    const r = await readPkcs12(fixture('rsa-legacy.p12'), 'secret')
    if (typeof r === 'string')
      throw new Error(r)
    expect(r.certs[0]!.subject).toBe('CN=rsa.example.test')
    expect(r.keys[0]!.matchesCert).toBe(0)
    expect(r.legacy).toBe(true)
    expect(r.encryption.join(' ')).toMatch(/RC2-40|3DES/)
  })
  it('reports a wrong password and garbage', async () => {
    expect(await readPkcs12(fixture('ec.p12'), 'wrong')).toBe('password')
    expect(await readPkcs12(new Uint8Array([1, 2, 3]), '')).toBe('invalid')
  })
})
