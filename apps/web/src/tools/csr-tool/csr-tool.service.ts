import type { JsonGeneralName } from '../certificate-decoder/x509'
import { isValidHost } from '@/utils/host'
import { parseIP } from '@/utils/ip'
import { csrFrom, describeKey, sha256Hex, toPem } from '@/utils/pki'
import { Pkcs10CertificateRequestGenerator, SubjectAlternativeNameExtension } from '../certificate-decoder/x509'

export interface DecodedCsr {
  subject: string
  sans: string[]
  publicKey: string
  signatureAlgorithm: string
  signatureValid: boolean
  spkiSha256: string
}

export async function decodeCsr(input: string): Promise<DecodedCsr | undefined> {
  const csr = csrFrom(input)
  if (!csr)
    return undefined
  // Look up by OID and re-parse: robust against duplicate class instances across bundles.
  const rawSan = csr.extensions.find(e => e.type === '2.5.29.17')
  const sanExt = rawSan ? new SubjectAlternativeNameExtension(rawSan.rawData) : undefined
  const sig = csr.signatureAlgorithm as Algorithm & { hash?: Algorithm }
  let publicKey = csr.publicKey.algorithm.name
  try {
    publicKey = describeKey((await csr.publicKey.export()).algorithm)
  }
  catch {}
  let signatureValid = false
  try {
    signatureValid = await csr.verify()
  }
  catch {}
  return {
    subject: csr.subject,
    sans: sanExt ? sanExt.names.items.map(n => (n.type === 'dns' ? n.value : `${n.type.toUpperCase()}:${n.value}`)) : [],
    publicKey,
    signatureAlgorithm: sig.hash ? `${sig.name} with ${sig.hash.name}` : sig.name,
    signatureValid,
    spkiSha256: await sha256Hex(csr.publicKey.rawData),
  }
}

export const KEY_TYPES = ['ec-p256', 'ec-p384', 'rsa-2048', 'rsa-3072', 'rsa-4096'] as const
export type KeyType = typeof KEY_TYPES[number]

export interface CsrSubject {
  cn: string
  o: string
  ou: string
  l: string
  st: string
  c: string
  email: string
}

export type SubjectError = 'cn' | 'c' | 'san' | 'email'

export function validateSubject(s: CsrSubject, sans: string[]): SubjectError | undefined {
  if (!s.cn.trim() || s.cn.length > 64)
    return 'cn'
  if (s.c && !/^[A-Z]{2}$/i.test(s.c.trim()))
    return 'c'
  if (s.email && !/^[^\s@]+@[^\s@]+$/.test(s.email.trim()))
    return 'email'
  if (sans.some(n => !parseIP(n) && !isValidHost(n.replace(/^\*\./, ''))))
    return 'san'
  return undefined
}

/** RFC 4514-style JSON name in the conventional order C, ST, L, O, OU, CN. */
function subjectName(s: CsrSubject): Record<string, string[]>[] {
  const parts: [string, string][] = [['C', s.c.toUpperCase()], ['ST', s.st], ['L', s.l], ['O', s.o], ['OU', s.ou], ['CN', s.cn], ['E', s.email]]
  return parts.filter(([, v]) => v.trim()).map(([k, v]) => ({ [k]: [v.trim()] }))
}

function keyParams(type: KeyType): { gen: RsaHashedKeyGenParams | EcKeyGenParams, sign: Algorithm | EcdsaParams } {
  if (type.startsWith('rsa')) {
    const gen: RsaHashedKeyGenParams = { name: 'RSASSA-PKCS1-v1_5', modulusLength: Number(type.slice(4)), publicExponent: new Uint8Array([1, 0, 1]), hash: 'SHA-256' }
    return { gen, sign: gen }
  }
  const p384 = type === 'ec-p384'
  return { gen: { name: 'ECDSA', namedCurve: p384 ? 'P-384' : 'P-256' }, sign: { name: 'ECDSA', hash: p384 ? 'SHA-384' : 'SHA-256' } as EcdsaParams }
}

/** Generates a key pair and a CSR in the browser. The private key never leaves the page. */
export async function generateCsr(subject: CsrSubject, sans: string[], type: KeyType): Promise<{ privateKeyPem: string, csrPem: string }> {
  const { gen, sign } = keyParams(type)
  const keys = await crypto.subtle.generateKey(gen, true, ['sign', 'verify']) as CryptoKeyPair
  const names: JsonGeneralName[] = sans.map(n => (parseIP(n) ? { type: 'ip', value: n } : { type: 'dns', value: n.toLowerCase() }))
  const csr = await Pkcs10CertificateRequestGenerator.create({
    name: subjectName(subject),
    keys,
    signingAlgorithm: sign,
    extensions: names.length ? [new SubjectAlternativeNameExtension(names)] : [],
  })
  return {
    privateKeyPem: toPem('PRIVATE KEY', await crypto.subtle.exportKey('pkcs8', keys.privateKey)),
    csrPem: csr.toString('pem'),
  }
}

/** SANs from free text; the CN is added first when it is a host name and missing (browsers ignore the CN). */
export function sanList(text: string, cn: string, includeCn: boolean): string[] {
  const list = text.split(/[\s,;]+/).map(s => s.trim()).filter(Boolean)
  const c = cn.trim()
  if (includeCn && c && (isValidHost(c.replace(/^\*\./, '')) || parseIP(c)) && !list.some(n => n.toLowerCase() === c.toLowerCase()))
    list.unshift(c)
  return [...new Set(list)]
}
