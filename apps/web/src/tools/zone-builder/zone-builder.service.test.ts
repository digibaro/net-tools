import { describe, expect, it } from 'vitest'
import { buildRecord, buildSpf, dateSerial, isValidName, rname, txtStrings, zoneHeader } from './zone-builder.service'

describe('isValidName', () => {
  it.each([
    ['@', true],
    ['www', true],
    ['_dmarc', true],
    ['*.example.com.', true],
    ['a.*.b', false],
    ['-bad', false],
    ['bad name', false],
    ['', false],
  ])('%s → %s', (n, want) => {
    expect(isValidName(n)).toBe(want)
  })
})

describe('txtStrings', () => {
  it('splits at 255 bytes and escapes', () => {
    const parts = txtStrings('x'.repeat(300))
    expect(parts).toHaveLength(2)
    expect(parts[0]).toBe(`"${'x'.repeat(255)}"`)
    expect(txtStrings('say "hi" \\ é')).toEqual(['"say \\"hi\\" \\\\ \\195\\169"'])
    expect(txtStrings('')).toEqual(['""'])
  })
})

describe('buildRecord', () => {
  it('formats common types', () => {
    expect(buildRecord({ kind: 'A', name: 'www', ttl: 300, address: '192.0.2.1' })).toEqual({ line: 'www\t300\tIN\tA\t192.0.2.1' })
    expect(buildRecord({ kind: 'AAAA', name: '@', address: '2001:0db8::0001' })).toEqual({ line: '@\tIN\tAAAA\t2001:db8::1' })
    expect(buildRecord({ kind: 'MX', name: '@', preference: 10, target: 'mail.example.com.' })).toEqual({ line: '@\tIN\tMX\t10 mail.example.com.' })
    expect(buildRecord({ kind: 'SRV', name: '@', service: 'sip', protocol: 'tcp', priority: 10, weight: 5, port: 5060, target: 'sip' })).toEqual({ line: '_sip._tcp\tIN\tSRV\t10 5 5060 sip' })
    expect(buildRecord({ kind: 'CAA', name: '@', critical: false, tag: 'issue', value: 'letsencrypt.org' })).toEqual({ line: '@\tIN\tCAA\t0 issue "letsencrypt.org"' })
    expect(buildRecord({ kind: 'SPF', name: '@', spf: { mx: true, a: false, includes: ['_spf.google.com'], ip4: [], ip6: [], all: '~' } })).toEqual({ line: '@\tIN\tTXT\t"v=spf1 mx include:_spf.google.com ~all"' })
  })
  it('reports invalid fields', () => {
    expect(buildRecord({ kind: 'A', name: 'www', address: '1.2.3' })).toEqual({ error: 'address' })
    expect(buildRecord({ kind: 'A', name: 'bad name', address: '1.2.3.4' })).toEqual({ error: 'name' })
    expect(buildRecord({ kind: 'MX', name: '@', preference: 70000, target: 'mx' })).toEqual({ error: 'number' })
    expect(buildRecord({ kind: 'CAA', name: '@', critical: false, tag: 'iodef', value: 'nope' })).toEqual({ error: 'value' })
    expect(buildRecord({ kind: 'A', name: '@', ttl: -1, address: '1.2.3.4' })).toEqual({ error: 'ttl' })
  })
})

describe('sPF and SOA', () => {
  it('builds SPF and counts lookups', () => {
    expect(buildSpf({ mx: true, a: true, includes: ['a.test', 'b.test'], ip4: ['192.0.2.0/24'], ip6: ['2001:db8::/32'], all: '-' })).toEqual({ record: 'v=spf1 ip4:192.0.2.0/24 ip6:2001:db8::/32 a mx include:a.test include:b.test -all', lookups: 4 })
    expect(buildSpf({ mx: false, a: false, includes: [], ip4: ['2001:db8::1'], ip6: [], all: '-' })).toEqual({ error: '2001:db8::1' })
  })
  it('builds the SOA header', () => {
    expect(dateSerial(new Date('2026-10-13T12:00:00Z'), 3)).toBe(2026101303)
    expect(rname('host.master@example.com')).toBe('host\\.master.example.com.')
    expect(rname('nope')).toBeUndefined()
    const h = zoneHeader({ origin: 'example.com', ttl: 3600, primary: 'ns1.example.com.', email: 'hostmaster@example.com', serial: 2026101301, refresh: 7200, retry: 3600, expire: 1209600, minimum: 3600 })!
    expect(h.split('\n').slice(0, 3)).toEqual(['$ORIGIN example.com.', '$TTL 3600', '@\tIN\tSOA\tns1.example.com. hostmaster.example.com. ('])
  })
})
