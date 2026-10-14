import type { ScanProbe } from './tls-scanner.service'
import { describe, expect, it } from 'vitest'
import { ciphersByVersion } from './tls-scanner.service'

const p = (version: string, name: string, weak: string[] = [], supported = true): ScanProbe => ({ kind: 'cipher', version, name, id: 0, supported, weak })

describe('ciphersByVersion', () => {
  it('groups supported ciphers, strongest first', () => {
    const g = ciphersByVersion([p('TLS 1.2', 'B_CBC', ['cbc']), p('TLS 1.2', 'A_GCM'), p('TLS 1.3', 'X'), p('TLS 1.0', 'Y', [], false)])
    expect(g.map(x => x.version)).toEqual(['TLS 1.3', 'TLS 1.2'])
    expect(g[1]!.ciphers.map(c => c.name)).toEqual(['A_GCM', 'B_CBC'])
  })
})
