import icmpData from './icmp.json'
import protocolData from './protocols.json'

export interface Protocol {
  number: number
  keyword: string
  name: string
  ipv6ExtensionHeader: boolean
  rfc: string[]
}

export interface IcmpType {
  type: number
  name: string
  rfc: string[]
  codes: { code: number, name: string }[]
}

export const PROTOCOLS = protocolData as Protocol[]
export const ICMP = icmpData as { v4: IcmpType[], v6: IcmpType[] }

const matches = (q: string, ...fields: (string | number)[]) => fields.some(f => String(f).toLowerCase().includes(q))

export function filterProtocols(query: string, list: Protocol[] = PROTOCOLS): Protocol[] {
  const q = query.trim().toLowerCase()
  if (!q)
    return list
  if (/^\d+$/.test(q))
    return list.filter(p => p.number === Number(q))
  return list.filter(p => matches(q, p.keyword, p.name))
}

export function filterIcmp(query: string, list: IcmpType[]): IcmpType[] {
  const q = query.trim().toLowerCase()
  if (!q)
    return list
  if (/^\d+$/.test(q))
    return list.filter(t => t.type === Number(q))
  return list.filter(t => matches(q, t.name) || t.codes.some(c => matches(q, c.name)))
}
