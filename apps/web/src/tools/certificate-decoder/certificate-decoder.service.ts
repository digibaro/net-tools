import type { Extension, GeneralName } from './x509'
import {
  AuthorityInfoAccessExtension,
  AuthorityKeyIdentifierExtension,
  BasicConstraintsExtension,
  CertificatePolicyExtension,
  CRLDistributionPointsExtension,
  ExtendedKeyUsageExtension,
  KeyUsageFlags,
  KeyUsagesExtension,
  SubjectAlternativeNameExtension,
  SubjectKeyIdentifierExtension,
  X509Certificate,
} from './x509'

const PEM_RE = /-----BEGIN CERTIFICATE-----[\s\S]+?-----END CERTIFICATE-----/g

const EKU_NAMES: Record<string, string> = {
  '1.3.6.1.5.5.7.3.1': 'TLS Web Server Authentication',
  '1.3.6.1.5.5.7.3.2': 'TLS Web Client Authentication',
  '1.3.6.1.5.5.7.3.3': 'Code Signing',
  '1.3.6.1.5.5.7.3.4': 'E-mail Protection',
  '1.3.6.1.5.5.7.3.8': 'Time Stamping',
  '1.3.6.1.5.5.7.3.9': 'OCSP Signing',
  '2.5.29.37.0': 'Any Extended Key Usage',
}

const POLICY_NAMES: Record<string, string> = {
  '2.23.140.1.2.1': 'Domain Validated (CA/B Forum)',
  '2.23.140.1.2.2': 'Organization Validated (CA/B Forum)',
  '2.23.140.1.2.3': 'Individual Validated (CA/B Forum)',
  '2.23.140.1.1': 'Extended Validation (CA/B Forum)',
}

export type Validity = 'valid' | 'expired' | 'not-yet-valid'

export interface DecodedCert {
  subject: string
  issuer: string
  serialNumber: string
  notBefore: string
  notAfter: string
  validity: Validity
  daysRemaining: number
  subjectAltNames: string[]
  publicKey: string
  signatureAlgorithm: string
  keyUsage: string[]
  extendedKeyUsage: string[]
  basicConstraints: string
  subjectKeyId: string
  authorityKeyId: string
  ocsp: string[]
  caIssuers: string[]
  crlDistributionPoints: string[]
  policies: string[]
  selfSigned: boolean
  sha256: string
  sha1: string
}

/** Splits pasted text into DER buffers: PEM blocks (chain) or a single base64 DER blob. */
export function extractCertificates(input: string): Uint8Array<ArrayBuffer>[] | undefined {
  const pems = input.match(PEM_RE)
  const blocks = pems ?? (input.trim() ? [input] : [])
  if (!blocks.length)
    return undefined
  try {
    return blocks.map((b) => {
      const b64 = b.replace(/-----(BEGIN|END) CERTIFICATE-----/g, '').replace(/\s+/g, '')
      if (!/^[A-Z0-9+/]+={0,2}$/i.test(b64))
        throw new Error('not base64')
      return Uint8Array.from(atob(b64), c => c.charCodeAt(0))
    })
  }
  catch {
    return undefined
  }
}

const hex = (buf: ArrayBuffer) => [...new Uint8Array(buf)].map(b => b.toString(16).padStart(2, '0').toUpperCase()).join(':')
const colonHex = (s: string) => s.toUpperCase().match(/.{1,2}/g)?.join(':') ?? ''
const generalName = (n: GeneralName) => (n.type === 'url' || n.type === 'dns' ? n.value : `${n.type.toUpperCase()}:${n.value}`)

function keyDescription(cert: X509Certificate): string {
  const alg = cert.publicKey.algorithm as Algorithm & { modulusLength?: number, namedCurve?: string }
  if (alg.modulusLength)
    return `RSA ${alg.modulusLength} bit`
  if (alg.namedCurve)
    return `${alg.name === 'ECDSA' || alg.name === 'ECDH' ? 'EC' : alg.name} ${alg.namedCurve}`
  return alg.name
}

function keyUsages(flags: KeyUsageFlags): string[] {
  return Object.entries(KeyUsageFlags)
    .filter(([, v]) => typeof v === 'number' && (flags & v) !== 0)
    .map(([k]) => k)
}

function ext<T extends Extension>(cert: X509Certificate, type: new (raw: BufferSource) => T): T | null {
  try {
    return cert.getExtension(type)
  }
  catch {
    return null
  }
}

export async function decodeCertificate(der: Uint8Array<ArrayBuffer>, now = new Date()): Promise<DecodedCert> {
  const cert = new X509Certificate(der)
  const san = ext(cert, SubjectAlternativeNameExtension)
  const ku = ext(cert, KeyUsagesExtension)
  const eku = ext(cert, ExtendedKeyUsageExtension)
  const bc = ext(cert, BasicConstraintsExtension)
  const ski = ext(cert, SubjectKeyIdentifierExtension)
  const aki = ext(cert, AuthorityKeyIdentifierExtension)
  const aia = ext(cert, AuthorityInfoAccessExtension)
  const crl = ext(cert, CRLDistributionPointsExtension)
  const pol = ext(cert, CertificatePolicyExtension)

  const validity: Validity = now < cert.notBefore ? 'not-yet-valid' : now > cert.notAfter ? 'expired' : 'valid'
  const sig = cert.signatureAlgorithm as Algorithm & { hash?: Algorithm }
  let selfSigned = false
  try {
    selfSigned = await cert.isSelfSigned()
  }
  catch {}

  return {
    subject: cert.subject,
    issuer: cert.issuer,
    serialNumber: colonHex(cert.serialNumber),
    notBefore: cert.notBefore.toISOString(),
    notAfter: cert.notAfter.toISOString(),
    validity,
    daysRemaining: Math.floor((cert.notAfter.getTime() - now.getTime()) / 86_400_000),
    subjectAltNames: san ? san.names.items.map(generalName) : [],
    publicKey: keyDescription(cert),
    signatureAlgorithm: sig.hash ? `${sig.name} with ${sig.hash.name}` : sig.name,
    keyUsage: ku ? keyUsages(ku.usages) : [],
    extendedKeyUsage: eku ? eku.usages.map(u => EKU_NAMES[String(u)] ?? String(u)) : [],
    basicConstraints: bc ? `CA: ${bc.ca ? 'yes' : 'no'}${bc.pathLength !== undefined ? `, path length ${bc.pathLength}` : ''}` : '',
    subjectKeyId: ski ? colonHex(ski.keyId) : '',
    authorityKeyId: aki?.keyId ? colonHex(aki.keyId) : '',
    ocsp: aia ? aia.ocsp.map(generalName) : [],
    caIssuers: aia ? aia.caIssuers.map(generalName) : [],
    crlDistributionPoints: crl ? crl.distributionPoints.flatMap(dp => dp.distributionPoint?.fullName?.map(n => n.uniformResourceIdentifier ?? '').filter(Boolean) ?? []) : [],
    policies: pol ? pol.policies.map(p => POLICY_NAMES[p] ? `${POLICY_NAMES[p]} (${p})` : p) : [],
    selfSigned,
    sha256: hex(await cert.getThumbprint('SHA-256')),
    sha1: hex(await cert.getThumbprint('SHA-1')),
  }
}

export async function decodeAll(input: string, now = new Date()): Promise<DecodedCert[] | undefined> {
  const ders = extractCertificates(input)
  if (!ders)
    return undefined
  try {
    return await Promise.all(ders.map(d => decodeCertificate(d, now)))
  }
  catch {
    return undefined
  }
}
