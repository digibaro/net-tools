import type { GatewayResult } from './gateway-info.service'
import { describe, expect, it } from 'vitest'
import { gatewayRows } from './gateway-info.service'

const r: GatewayResult = {
  interface: 'eth0',
  gateway: '192.168.1.1',
  gatewayMac: 'aa:bb:cc:dd:ee:01',
  addresses: ['192.168.1.20/24'],
  externalIp: '100.64.1.2',
  doubleNat: true,
  natPmp: { supported: false, externalIp: '', epoch: 0, pcp: false, error: 'no answer' },
  upnp: { supported: true, externalIp: '100.64.1.2', friendlyName: 'Router', model: 'RT', service: 'x', error: '' },
}

describe('gatewayRows', () => {
  it('summarizes each protocol', () => {
    const rows = gatewayRows(r, k => k)
    expect(rows.find(x => x.label === 'NAT-PMP')?.value).toBe('no answer')
    expect(rows.find(x => x.label === 'UPnP IGD')?.value).toBe('100.64.1.2 — Router · RT')
    expect(rows.find(x => x.label === 'rows.externalIp')?.value).toBe('100.64.1.2')
  })
})
