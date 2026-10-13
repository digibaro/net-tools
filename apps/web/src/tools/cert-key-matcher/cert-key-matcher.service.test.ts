import { describe, expect, it } from 'vitest'
import { toPem } from '@/utils/pki'
import { Pkcs10CertificateRequestGenerator, X509CertificateGenerator } from '../certificate-decoder/x509'
import { compare } from './cert-key-matcher.service'

const alg = { name: 'ECDSA', namedCurve: 'P-256', hash: 'SHA-256' }

async function material() {
  const keys = await crypto.subtle.generateKey(alg, true, ['sign', 'verify']) as CryptoKeyPair
  const cert = await X509CertificateGenerator.createSelfSigned({ serialNumber: '01', name: 'CN=test.example', notBefore: new Date(), notAfter: new Date(Date.now() + 86_400_000), keys, signingAlgorithm: alg })
  const csr = await Pkcs10CertificateRequestGenerator.create({ name: 'CN=test.example', keys, signingAlgorithm: alg })
  return { certificate: cert.toString('pem'), csr: csr.toString('pem'), key: toPem('PRIVATE KEY', await crypto.subtle.exportKey('pkcs8', keys.privateKey)) }
}

describe('compare', () => {
  it('matches a certificate, key and CSR from one key pair', async () => {
    const m = await material()
    const r = await compare(m)
    expect(r.match).toBe(true)
    expect(r.items.map(i => i.description)).toEqual(['EC P-256', 'EC P-256', 'EC P-256'])
    expect(r.items[0]!.subject).toBe('CN=test.example')
  })
  it('detects a mismatch', async () => {
    const a = await material()
    const b = await material()
    expect((await compare({ certificate: a.certificate, key: b.key, csr: '' })).match).toBe(false)
  })
  it('needs two readable items', async () => {
    const a = await material()
    const r = await compare({ certificate: a.certificate, key: 'garbage', csr: '' })
    expect(r.match).toBeUndefined()
    expect(r.items[1]).toMatchObject({ present: true, error: 'no-key' })
    expect(r.items[2]).toEqual({ kind: 'csr', present: false })
  })
})
