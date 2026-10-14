import type { PortEntry } from './port-reference.service'
import { describe, expect, it } from 'vitest'
import { searchPorts } from './port-reference.service'

const DATA: PortEntry[] = [
  ['22', 'tcp', 'ssh', 'The Secure Shell (SSH) Protocol', 'RFC 4251'],
  ['443', 'tcp', 'https', 'http protocol over TLS/SSL', 'RFC 9110'],
  ['443', 'udp', 'https', 'http protocol over TLS/SSL', 'RFC 9110'],
  ['6000-6063', 'tcp', 'x11', 'X Window System', ''],
  ['8443', 'tcp', 'pcsync-https', 'PCsync HTTPS', ''],
]

describe('searchPorts', () => {
  it('finds by port number, including ranges', () => {
    expect(searchPorts(DATA, '443', 'all').rows.map(r => r.protocol)).toEqual(['tcp', 'udp'])
    expect(searchPorts(DATA, '6010', 'all').rows[0]?.service).toBe('x11')
  })

  it('filters by protocol', () => {
    expect(searchPorts(DATA, '443', 'udp').total).toBe(1)
  })

  it('ranks exact service names first, then partial matches', () => {
    expect(searchPorts(DATA, 'https', 'tcp').rows.map(r => r.port)).toEqual(['443', '8443'])
    expect(searchPorts(DATA, 'secure shell', 'all').rows[0]?.port).toBe('22')
  })

  it('returns nothing for an empty query', () => {
    expect(searchPorts(DATA, ' ', 'all')).toEqual({ rows: [], total: 0 })
  })
})

describe('bundled data', async () => {
  const { loadPorts } = await import('./port-reference.service')
  it('contains well-known ports', async () => {
    const ports = await loadPorts()
    expect(searchPorts(ports, '53', 'udp').rows[0]?.service).toBe('domain')
    expect(ports.length).toBeGreaterThan(5000)
  })
})
