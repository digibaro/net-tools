import { describe, expect, it } from 'vitest'
import { decodeCsr, generateCsr, sanList, validateSubject } from './csr-tool.service'

const subject = { cn: 'www.example.com', o: 'Example Ltd', ou: '', l: 'Amsterdam', st: '', c: 'NL', email: '' }

describe('generate and decode', () => {
  it.each(['ec-p256', 'rsa-2048'] as const)('%s', async (type) => {
    const { privateKeyPem, csrPem } = await generateCsr(subject, ['www.example.com', 'example.com', '192.0.2.1'], type)
    expect(privateKeyPem).toMatch(/^-----BEGIN PRIVATE KEY-----/)
    const d = (await decodeCsr(csrPem))!
    expect(d.signatureValid).toBe(true)
    expect(d.subject).toContain('CN=www.example.com')
    expect(d.subject).toContain('C=NL')
    expect(d.sans).toEqual(['www.example.com', 'example.com', 'IP:192.0.2.1'])
    expect(d.publicKey).toBe(type === 'ec-p256' ? 'EC P-256' : 'RSA 2048 bit')
  })
  it('rejects non-CSR input', async () => {
    expect(await decodeCsr('hello')).toBeUndefined()
  })
})

describe('helpers', () => {
  it('builds SAN lists', () => {
    expect(sanList('example.com, www.example.com', 'www.example.com', true)).toEqual(['example.com', 'www.example.com'])
    expect(sanList('example.com', 'www.example.com', true)).toEqual(['www.example.com', 'example.com'])
    expect(sanList('', 'My Server', true)).toEqual([])
  })
  it('validates the subject', () => {
    expect(validateSubject(subject, ['a.example.com'])).toBeUndefined()
    expect(validateSubject({ ...subject, c: 'NLD' }, [])).toBe('c')
    expect(validateSubject({ ...subject, cn: '' }, [])).toBe('cn')
    expect(validateSubject(subject, ['bad host'])).toBe('san')
  })
})
