import type { KeyError } from '@/utils/pki'
import { certificateFrom, csrFrom, privateKeySpki, publicKeyDescription, sha256Hex } from '@/utils/pki'

export type ItemKind = 'certificate' | 'key' | 'csr'

export interface Item {
  kind: ItemKind
  /** Empty input: the item takes no part in the comparison. */
  present: boolean
  error?: 'invalid' | KeyError
  description?: string
  subject?: string
  spki?: string
}

export interface Comparison {
  items: Item[]
  /** undefined when fewer than two items could be read. */
  match?: boolean
}

async function readItem(kind: ItemKind, input: string): Promise<Item> {
  if (!input.trim())
    return { kind, present: false }
  if (kind === 'key') {
    const r = await privateKeySpki(input)
    return typeof r === 'string' ? { kind, present: true, error: r } : { kind, present: true, description: r.description, spki: await sha256Hex(r.spki) }
  }
  const obj = kind === 'certificate' ? certificateFrom(input) : csrFrom(input)
  if (!obj)
    return { kind, present: true, error: 'invalid' }
  return { kind, present: true, subject: obj.subject, description: await publicKeyDescription(obj.publicKey), spki: await sha256Hex(obj.publicKey.rawData) }
}

/** Compares the public keys (SPKI SHA-256) of a certificate, a private key and a CSR. */
export async function compare(inputs: Record<ItemKind, string>): Promise<Comparison> {
  const items = await Promise.all((['certificate', 'key', 'csr'] as const).map(k => readItem(k, inputs[k])))
  const hashes = items.filter(i => i.spki).map(i => i.spki!)
  return { items, match: hashes.length >= 2 ? hashes.every(h => h === hashes[0]) : undefined }
}
