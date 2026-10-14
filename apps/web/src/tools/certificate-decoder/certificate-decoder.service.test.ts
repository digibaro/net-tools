import { describe, expect, it } from 'vitest'
import { decodeAll, extractCertificates } from './certificate-decoder.service'
import { CA_PEM, LEAF_DER_BASE64, LEAF_PEM } from './testdata/certs'

const NOW = new Date('2027-01-01T00:00:00Z')

describe('certificate decoder', () => {
  it('decodes a leaf certificate (expected values from openssl x509)', async () => {
    const [leaf] = (await decodeAll(LEAF_PEM, NOW))!
    expect(leaf).toMatchObject({
      subject: 'CN=www.example.test',
      issuer: 'C=NL, O=NetTools Test, CN=NetTools Test CA',
      serialNumber: '12:34:AB:CD',
      notBefore: '2026-01-01T00:00:00.000Z',
      notAfter: '2028-04-05T00:00:00.000Z',
      validity: 'valid',
      daysRemaining: 460,
      subjectAltNames: ['www.example.test', 'example.test', 'IP:192.0.2.10'],
      publicKey: 'RSA 2048 bit',
      keyUsage: ['digitalSignature', 'keyEncipherment'],
      extendedKeyUsage: ['TLS Web Server Authentication', 'TLS Web Client Authentication'],
      basicConstraints: 'CA: no',
      ocsp: ['http://ocsp.example.test'],
      caIssuers: ['http://ca.example.test/ca.crt'],
      crlDistributionPoints: ['http://crl.example.test/ca.crl'],
      policies: ['Domain Validated (CA/B Forum) (2.23.140.1.2.1)'],
      selfSigned: false,
      sha256: 'BE:29:C5:4F:1B:3E:E7:6D:A9:90:C4:06:12:47:49:D4:E1:10:04:EF:46:DF:25:BC:FC:EF:19:7C:02:2D:2E:FF',
      sha1: '3D:2D:87:1C:9E:23:FF:96:E1:0E:40:AC:10:8B:1A:BA:40:03:7E:E1',
    })
    expect(leaf!.signatureAlgorithm).toMatch(/^ECDSA with SHA-256$/)
    expect(leaf!.authorityKeyId).toMatch(/^([0-9A-F]{2}:){19}[0-9A-F]{2}$/)
  })

  it('decodes a chain and recognizes the self-signed CA', async () => {
    const certs = (await decodeAll(`${LEAF_PEM}\n${CA_PEM}`, NOW))!
    expect(certs).toHaveLength(2)
    expect(certs[1]).toMatchObject({ selfSigned: true, publicKey: 'EC P-256', basicConstraints: 'CA: yes, path length 0', keyUsage: ['keyCertSign', 'cRLSign'] })
  })

  it('accepts bare base64 DER and reports validity', async () => {
    expect((await decodeAll(LEAF_DER_BASE64, new Date('2030-01-01')))![0]!.validity).toBe('expired')
    expect((await decodeAll(LEAF_DER_BASE64, new Date('2025-01-01')))![0]!.validity).toBe('not-yet-valid')
  })

  it('rejects garbage', async () => {
    expect(extractCertificates('')).toBeUndefined()
    expect(extractCertificates('not base64 !!')).toBeUndefined()
    await expect(decodeAll('-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----')).resolves.toBeUndefined()
  })
})
