import { formatIP, parseIP, splitList } from '@/utils/ip'
import { classify } from '@/utils/special-ranges'

export interface CheckRow {
  ip: string
  range: string
  name: string
  rfc: string
  globallyReachable: boolean
  bogon: boolean
}

export function checkIps(input: string): { rows: CheckRow[], invalid: string[] } {
  const rows: CheckRow[] = []
  const invalid: string[] = []
  for (const s of splitList(input).slice(0, 5000)) {
    const ip = parseIP(s)
    if (!ip) {
      invalid.push(s)
      continue
    }
    const c = classify(ip)
    const p = c.primary
    rows.push({
      ip: formatIP(ip.version, ip.value),
      range: p?.cidr ?? (c.unallocated ? '—' : ip.version === 4 ? '—' : '2000::/3'),
      name: p?.name ?? (c.unallocated ? 'Unallocated / reserved by IETF' : ip.version === 6 ? 'Global Unicast' : 'Public'),
      rfc: p?.rfc ?? '',
      globallyReachable: !c.bogon,
      bogon: c.bogon,
    })
  }
  return { rows, invalid }
}
