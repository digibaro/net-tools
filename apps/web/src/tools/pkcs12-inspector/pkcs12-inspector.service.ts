import type forgeNs from 'node-forge'
import { X509Certificate } from '@/tools/certificate-decoder/x509'
import { privateKeySpki, toPem } from '@/utils/pki'

type Forge = typeof forgeNs
type Asn1 = forgeNs.asn1.Asn1

export interface P12Cert { pem: string, subject: string, issuer: string, notBefore: string, notAfter: string, sha256: string, friendlyName: string, localKeyId: string, selfSigned: boolean }
export interface P12Key { pem: string, description: string, friendlyName: string, localKeyId: string, matchesCert: number }
export interface P12Result {
  certs: P12Cert[]
  keys: P12Key[]
  mac: string
  encryption: string[]
  legacy: boolean
}
export type P12Error = 'password' | 'invalid'

const OIDS: Record<string, string> = {
  '1.2.840.113549.1.12.1.1': 'RC4-128',
  '1.2.840.113549.1.12.1.2': 'RC4-40',
  '1.2.840.113549.1.12.1.3': '3DES (pbeWithSHAAnd3-KeyTripleDES-CBC)',
  '1.2.840.113549.1.12.1.5': 'RC2-128',
  '1.2.840.113549.1.12.1.6': 'RC2-40 (pbeWithSHAAnd40BitRC2-CBC)',
  '1.2.840.113549.1.5.13': 'PBES2',
  '1.2.840.113549.1.5.12': 'PBKDF2',
  '2.16.840.1.101.3.4.1.2': 'AES-128-CBC',
  '2.16.840.1.101.3.4.1.22': 'AES-192-CBC',
  '2.16.840.1.101.3.4.1.42': 'AES-256-CBC',
  '1.2.840.113549.3.7': '3DES-CBC',
  '1.2.840.113549.2.7': 'hmacWithSHA1',
  '1.2.840.113549.2.9': 'hmacWithSHA256',
  '1.2.840.113549.2.10': 'hmacWithSHA384',
  '1.2.840.113549.2.11': 'hmacWithSHA512',
  '1.3.14.3.2.26': 'SHA-1',
  '2.16.840.1.101.3.4.2.1': 'SHA-256',
  '2.16.840.1.101.3.4.2.2': 'SHA-384',
  '2.16.840.1.101.3.4.2.3': 'SHA-512',
}
const LEGACY = ['RC4', 'RC2', '3DES', 'SHA-1 ']

const bytesToBinary = (b: Uint8Array) => Array.from(b, c => String.fromCharCode(c)).join('')
const binaryToBytes = (s: string) => Uint8Array.from(s, c => c.charCodeAt(0))
const hex = (s: string) => Array.from(s, c => c.charCodeAt(0).toString(16).padStart(2, '0')).join('')

function int(forge: Forge, a: Asn1): number {
  return forge.util.createBuffer(a.value as string).getInt((a.value as string).length * 8)
}

/** "PBES2 · PBKDF2 hmacWithSHA256, 2048 iterations · AES-256-CBC" or "3DES (…), 2048 iterations". */
function describeAlg(forge: Forge, alg: Asn1): string {
  const v = alg.value as Asn1[]
  const oid = forge.asn1.derToOid(v[0]!.value as string)
  const name = OIDS[oid] ?? oid
  const params = v[1]?.value as Asn1[] | undefined
  if (name === 'PBES2' && params) {
    const kdf = params[0]!.value as Asn1[]
    const kdfParams = kdf[1]!.value as Asn1[]
    const iter = int(forge, kdfParams[1]!)
    const prf = kdfParams.find(p => Array.isArray(p.value)) // AlgorithmIdentifier of the PRF (optional, default SHA-1)
    const prfName = prf ? OIDS[forge.asn1.derToOid((prf.value as Asn1[])[0]!.value as string)] : 'hmacWithSHA1'
    const enc = forge.asn1.derToOid((params[1]!.value as Asn1[])[0]!.value as string)
    return `PBES2 · PBKDF2 ${prfName}, ${iter} iterations · ${OIDS[enc] ?? enc}`
  }
  if (params && params.length > 1)
    return `${name}, ${int(forge, params[1]!)} iterations`
  return name
}

function contentOf(forge: Forge, node: Asn1): Asn1 {
  // [0] EXPLICIT { OCTET STRING } — the octet string holds DER (possibly split into constructed parts)
  const inner = (node.value as Asn1[])[0]!
  const raw = typeof inner.value === 'string' ? inner.value : (inner.value as Asn1[]).map(p => p.value as string).join('')
  return forge.asn1.fromDer(raw)
}

/** Algorithms used for the certificate bags, shrouded keys and the MAC. */
function algorithms(forge: Forge, pfx: Asn1): { mac: string, encryption: string[] } {
  const top = pfx.value as Asn1[]
  let mac = 'none'
  if (top[2]) {
    const macData = top[2].value as Asn1[]
    const digestAlg = ((macData[0]!.value as Asn1[])[0]!.value as Asn1[])[0]!
    const iterations = macData[2] ? int(forge, macData[2]) : 1
    mac = `${OIDS[forge.asn1.derToOid(digestAlg.value as string)] ?? 'unknown'}, ${iterations} iterations`
  }
  const enc = new Set<string>()
  const authSafe = contentOf(forge, (top[1]!.value as Asn1[])[1]!)
  for (const ci of authSafe.value as Asn1[]) {
    const [type, content] = ci.value as Asn1[]
    const oid = forge.asn1.derToOid(type!.value as string)
    if (oid === '1.2.840.113549.1.7.6') { // encryptedData: certificates are usually here
      const encInfo = ((content!.value as Asn1[])[0]!.value as Asn1[])[1]!
      enc.add(`certificates: ${describeAlg(forge, (encInfo.value as Asn1[])[1]!)}`)
    }
    else if (oid === '1.2.840.113549.1.7.1') {
      for (const bag of contentOf(forge, content!).value as Asn1[]) {
        const [bagId, bagValue] = bag.value as Asn1[]
        if (forge.asn1.derToOid(bagId!.value as string) === forge.pki.oids.pkcs8ShroudedKeyBag)
          enc.add(`private key: ${describeAlg(forge, ((bagValue!.value as Asn1[])[0]!.value as Asn1[])[0]!)}`)
      }
    }
  }
  return { mac, encryption: [...enc] }
}

export async function readPkcs12(bytes: Uint8Array, password: string): Promise<P12Result | P12Error> {
  const forge = (await import('node-forge')).default
  let pfx: Asn1
  try {
    pfx = forge.asn1.fromDer(bytesToBinary(bytes))
  }
  catch {
    return 'invalid'
  }
  let p12: forgeNs.pkcs12.Pkcs12Pfx
  try {
    p12 = forge.pkcs12.pkcs12FromAsn1(pfx, false, password)
  }
  catch (e) {
    return /password|MAC|decrypt/i.test(String(e)) ? 'password' : 'invalid'
  }
  const attr = (bag: forgeNs.pkcs12.Bag, name: 'friendlyName' | 'localKeyId') => {
    const v = bag.attributes?.[name]?.[0] as string | undefined
    return v === undefined ? '' : name === 'localKeyId' ? hex(v) : v
  }
  const certs: (P12Cert & { spki: Uint8Array })[] = []
  const certBag = forge.pki.oids.certBag!
  for (const bag of p12.getBags({ bagType: certBag })[certBag] ?? []) {
    const der = forge.asn1.toDer(bag.cert ? forge.pki.certificateToAsn1(bag.cert) : bag.asn1!).getBytes()
    const c = new X509Certificate(binaryToBytes(der))
    certs.push({
      pem: c.toString('pem'),
      subject: c.subject,
      issuer: c.issuer,
      notBefore: c.notBefore.toISOString(),
      notAfter: c.notAfter.toISOString(),
      sha256: [...new Uint8Array(await c.getThumbprint('SHA-256'))].map(b => b.toString(16).padStart(2, '0')).join(':').toUpperCase(),
      friendlyName: attr(bag, 'friendlyName'),
      localKeyId: attr(bag, 'localKeyId'),
      selfSigned: c.subject === c.issuer,
      spki: new Uint8Array(c.publicKey.rawData),
    })
  }
  const keys: P12Key[] = []
  for (const type of [forge.pki.oids.pkcs8ShroudedKeyBag!, forge.pki.oids.keyBag!]) {
    for (const bag of p12.getBags({ bagType: type })[type] ?? []) {
      const info = bag.key ? forge.pki.wrapRsaPrivateKey(forge.pki.privateKeyToAsn1(bag.key)) : bag.asn1!
      const pem = toPem('PRIVATE KEY', binaryToBytes(forge.asn1.toDer(info).getBytes()))
      const k = await privateKeySpki(pem)
      const spki = typeof k === 'string' ? undefined : new Uint8Array(k.spki)
      keys.push({
        pem,
        description: typeof k === 'string' ? k : k.description,
        friendlyName: attr(bag, 'friendlyName'),
        localKeyId: attr(bag, 'localKeyId'),
        matchesCert: spki ? certs.findIndex(c => c.spki.length === spki.length && c.spki.every((b, i) => b === spki[i])) : -1,
      })
    }
  }
  const { mac, encryption } = algorithms(forge, pfx)
  const legacy = [...encryption, `${mac} `].some(e => LEGACY.some(l => e.includes(l)))
  return { certs: certs.map(({ spki: _, ...c }) => c), keys, mac, encryption, legacy }
}
