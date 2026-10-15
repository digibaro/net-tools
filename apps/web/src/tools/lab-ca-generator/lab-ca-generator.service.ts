import * as x509 from '@/tools/certificate-decoder/x509'
import { parseIP } from '@/utils/ip'
import { pemBlocks, toPem } from '@/utils/pki'

export const KEY_TYPES = ['ecdsa-p256', 'ecdsa-p384', 'rsa-2048', 'rsa-3072', 'rsa-4096'] as const
export type KeyType = typeof KEY_TYPES[number]

export interface LabOptions {
  caName: string
  organization: string
  keyType: KeyType
  caDays: number
  serverName: string
  /** DNS names and IP addresses; the server name is always included. */
  sans: string[]
  serverDays: number
  clientAuth: boolean
}

export interface LabResult {
  caCert: string
  caKey: string
  serverCert: string
  serverKey: string
  fullchain: string
}

const EKU_SERVER = '1.3.6.1.5.5.7.3.1'
const EKU_CLIENT = '1.3.6.1.5.5.7.3.2'

function algorithms(k: KeyType): { gen: EcKeyGenParams | RsaHashedKeyGenParams, sign: EcdsaParams | Algorithm } {
  if (k.startsWith('ecdsa')) {
    const curve = k === 'ecdsa-p384' ? 'P-384' : 'P-256'
    const hash = curve === 'P-384' ? 'SHA-384' : 'SHA-256'
    return { gen: { name: 'ECDSA', namedCurve: curve }, sign: { name: 'ECDSA', hash } as EcdsaParams }
  }
  const modulusLength = Number(k.slice(4))
  return { gen: { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-256', publicExponent: new Uint8Array([1, 0, 1]), modulusLength }, sign: { name: 'RSASSA-PKCS1-v1_5' } }
}

/** A SAN entry is a DNS name (wildcards allowed) or an IPv4/IPv6 address. */
export function sanType(s: string): 'ip' | 'dns' | undefined {
  const v = s.trim()
  if (parseIP(v))
    return 'ip'
  return /^(?:\*\.)?(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/i.test(v) && v.length <= 253 ? 'dns' : undefined
}

/** CN values become part of an X.500 name: escape the RFC 4514 special characters. */
const rdn = (s: string) => s.trim().replace(/[,+"\\<>;=]/g, c => `\\${c}`)

function serial(): string {
  const b = crypto.getRandomValues(new Uint8Array(16))
  b[0]! &= 0x7F // positive INTEGER
  return [...b].map(x => x.toString(16).padStart(2, '0')).join('')
}

const days = (n: number) => new Date(Date.now() + n * 86400_000)

async function keyPem(k: CryptoKey): Promise<string> {
  return toPem('PRIVATE KEY', await crypto.subtle.exportKey('pkcs8', k))
}

export interface ExistingCA { certPem: string, keyPem: string }
export type CAError = 'ca-cert' | 'ca-key' | 'ca-mismatch' | 'ca-not-ca'

/** Loads an existing CA (certificate + PKCS#8 key) and checks that they belong together. */
export async function loadCA(e: ExistingCA): Promise<{ cert: x509.X509Certificate, key: CryptoKey, sign: EcdsaParams | Algorithm } | CAError> {
  let cert: x509.X509Certificate
  try {
    cert = new x509.X509Certificate(e.certPem.trim())
  }
  catch {
    return 'ca-cert'
  }
  if (!cert.getExtension(x509.BasicConstraintsExtension)?.ca)
    return 'ca-not-ca'
  const der = pemBlocks(e.keyPem).find(b => b.label === 'PRIVATE KEY')?.der
  if (!der)
    return 'ca-key'
  const alg = cert.publicKey.algorithm as EcKeyAlgorithm | RsaHashedKeyAlgorithm
  const isEc = alg.name === 'ECDSA'
  const importAlg = isEc ? { name: 'ECDSA', namedCurve: (alg as EcKeyAlgorithm).namedCurve } : { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-256' }
  let key: CryptoKey
  try {
    key = await crypto.subtle.importKey('pkcs8', der, importAlg, true, ['sign'])
  }
  catch {
    return 'ca-key'
  }
  const sign = isEc ? { name: 'ECDSA', hash: (alg as EcKeyAlgorithm).namedCurve === 'P-384' ? 'SHA-384' : 'SHA-256' } as EcdsaParams : { name: 'RSASSA-PKCS1-v1_5' }
  // The key matches when it can sign something the certificate's public key verifies.
  const probe = new Uint8Array([1, 2, 3])
  const sig = await crypto.subtle.sign(isEc ? sign : importAlg, key, probe)
  const pub = await cert.publicKey.export(isEc ? importAlg : { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-256' }, ['verify'])
  if (!await crypto.subtle.verify(isEc ? sign : importAlg, pub, sig, probe))
    return 'ca-mismatch'
  return { cert, key, sign }
}

/** Generates a lab root CA (or reuses `existing`) and one server certificate signed by it. */
export async function generateLab(o: LabOptions, existing?: ExistingCA): Promise<LabResult | CAError> {
  const { gen, sign: newSign } = algorithms(o.keyType)
  const notBefore = new Date(Date.now() - 60_000)
  let ca: x509.X509Certificate
  let caPrivate: CryptoKey
  let caPublic: x509.PublicKeyType
  let sign = newSign
  let caKeyPem = ''
  if (existing) {
    const loaded = await loadCA(existing)
    if (typeof loaded === 'string')
      return loaded
    ca = loaded.cert
    caPrivate = loaded.key
    caPublic = loaded.cert.publicKey
    sign = loaded.sign
    caKeyPem = existing.keyPem.trim()
  }
  else {
    const caKeys = await crypto.subtle.generateKey(gen, true, ['sign', 'verify']) as CryptoKeyPair
    const caSubject = `CN=${rdn(o.caName)}${o.organization.trim() ? `, O=${rdn(o.organization)}` : ''}`
    ca = await x509.X509CertificateGenerator.createSelfSigned({
      serialNumber: serial(),
      name: caSubject,
      notBefore,
      notAfter: days(o.caDays),
      signingAlgorithm: sign,
      keys: caKeys,
      extensions: [
        new x509.BasicConstraintsExtension(true, 0, true),
        new x509.KeyUsagesExtension(x509.KeyUsageFlags.keyCertSign | x509.KeyUsageFlags.cRLSign | x509.KeyUsageFlags.digitalSignature, true),
        await x509.SubjectKeyIdentifierExtension.create(caKeys.publicKey),
      ],
    })
    caPrivate = caKeys.privateKey
    caPublic = caKeys.publicKey
    caKeyPem = await keyPem(caKeys.privateKey)
  }
  const srvKeys = await crypto.subtle.generateKey(gen, true, ['sign', 'verify']) as CryptoKeyPair
  const names = [...new Set([o.serverName.trim(), ...o.sans.map(s => s.trim())].filter(Boolean))]
  const usage = o.keyType.startsWith('rsa') ? x509.KeyUsageFlags.digitalSignature | x509.KeyUsageFlags.keyEncipherment : x509.KeyUsageFlags.digitalSignature
  const server = await x509.X509CertificateGenerator.create({
    serialNumber: serial(),
    subject: `CN=${rdn(o.serverName)}`,
    issuer: ca.subject,
    notBefore,
    notAfter: new Date(Math.min(days(o.serverDays).getTime(), existing ? ca.notAfter.getTime() : days(o.caDays).getTime())),
    signingAlgorithm: sign,
    publicKey: srvKeys.publicKey,
    signingKey: caPrivate,
    extensions: [
      new x509.BasicConstraintsExtension(false, undefined, true),
      new x509.KeyUsagesExtension(usage, true),
      new x509.ExtendedKeyUsageExtension(o.clientAuth ? [EKU_SERVER, EKU_CLIENT] : [EKU_SERVER]),
      new x509.SubjectAlternativeNameExtension(names.map(n => ({ type: sanType(n) === 'ip' ? 'ip' as const : 'dns' as const, value: n }))),
      await x509.SubjectKeyIdentifierExtension.create(srvKeys.publicKey),
      await x509.AuthorityKeyIdentifierExtension.create(caPublic),
    ],
  })
  const caCert = ca.toString('pem')
  const serverCert = server.toString('pem')
  return { caCert, caKey: caKeyPem, serverCert, serverKey: await keyPem(srvKeys.privateKey), fullchain: `${serverCert}\n${caCert}\n` }
}
