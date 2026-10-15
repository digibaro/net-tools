import type { LabResult } from './lab-ca-generator.service'
import { describe, expect, it } from 'vitest'
import * as x509 from '@/tools/certificate-decoder/x509'
import { generateLab, sanType } from './lab-ca-generator.service'

describe('lab-ca-generator', () => {
  it('classifies SAN entries', () => {
    expect(sanType('*.lab.example')).toBe('dns')
    expect(sanType('192.168.1.10')).toBe('ip')
    expect(sanType('fd00::1')).toBe('ip')
    expect(sanType('bad name')).toBeUndefined()
  })
  it('generates a CA and a server certificate it signed', async () => {
    const r = await generateLab({ caName: 'Lab Root, test', organization: 'Home', keyType: 'ecdsa-p256', caDays: 3650, serverName: 'nas.lab.example', sans: ['192.168.1.10', 'nas'], serverDays: 825, clientAuth: true }) as LabResult
    const ca = new x509.X509Certificate(r.caCert)
    const srv = new x509.X509Certificate(r.serverCert)
    expect(ca.subject).toBe('CN=Lab Root\\, test, O=Home')
    expect(srv.issuer).toBe(ca.subject)
    expect(await srv.verify({ publicKey: ca.publicKey, signatureOnly: true })).toBe(true)
    expect(await ca.isSelfSigned()).toBe(true)
    const san = srv.getExtension(x509.SubjectAlternativeNameExtension)!
    expect(san.names.items.map(n => `${n.type}:${n.value}`)).toEqual(['dns:nas.lab.example', 'ip:192.168.1.10', 'dns:nas'])
    expect(srv.getExtension(x509.BasicConstraintsExtension)!.ca).toBe(false)
    expect(srv.getExtension(x509.ExtendedKeyUsageExtension)!.usages).toEqual(['1.3.6.1.5.5.7.3.1', '1.3.6.1.5.5.7.3.2'])
    expect(r.serverKey).toMatch(/^-----BEGIN PRIVATE KEY-----/)
    expect(r.fullchain.match(/BEGIN CERTIFICATE/g)).toHaveLength(2)
  })
  it('supports RSA', async () => {
    const r = await generateLab({ caName: 'Lab', organization: '', keyType: 'rsa-2048', caDays: 30, serverName: 'a.example', sans: [], serverDays: 60, clientAuth: false }) as LabResult
    const srv = new x509.X509Certificate(r.serverCert)
    expect(srv.notAfter.getTime()).toBeLessThanOrEqual(new x509.X509Certificate(r.caCert).notAfter.getTime())
  }, 30_000)
  it('issues more certificates from an existing CA', async () => {
    const opts = { caName: 'Lab', organization: '', keyType: 'ecdsa-p256' as const, caDays: 3650, serverName: 'a.lab', sans: [], serverDays: 90, clientAuth: false }
    const first = await generateLab(opts) as LabResult
    const second = await generateLab({ ...opts, serverName: 'b.lab' }, { certPem: first.caCert, keyPem: first.caKey }) as LabResult
    const ca = new x509.X509Certificate(first.caCert)
    const srv = new x509.X509Certificate(second.serverCert)
    expect(second.caCert).toBe(first.caCert)
    expect(srv.subject).toBe('CN=b.lab')
    expect(await srv.verify({ publicKey: ca.publicKey, signatureOnly: true })).toBe(true)
    const other = await generateLab(opts) as LabResult
    expect(await generateLab(opts, { certPem: first.caCert, keyPem: other.caKey })).toBe('ca-mismatch')
    expect(await generateLab(opts, { certPem: first.serverCert, keyPem: first.serverKey })).toBe('ca-not-ca')
    expect(await generateLab(opts, { certPem: 'x', keyPem: 'y' })).toBe('ca-cert')
  })
})
