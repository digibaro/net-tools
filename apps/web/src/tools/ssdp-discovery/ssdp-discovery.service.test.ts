import type { SsdpDevice } from './ssdp-discovery.service'
import { describe, expect, it } from 'vitest'
import { label, shortUrn } from './ssdp-discovery.service'

describe('ssdp-discovery', () => {
  it('shortens URNs', () => {
    expect(shortUrn('urn:schemas-upnp-org:device:InternetGatewayDevice:1')).toBe('InternetGatewayDevice')
    expect(shortUrn('urn:dial-multiscreen-org:service:dial:1')).toBe('dial')
    expect(shortUrn('upnp:rootdevice')).toBe('upnp:rootdevice')
  })
  it('labels devices', () => {
    expect(label({ friendlyName: '', model: '', server: 'MiniUPnPd', address: '192.168.1.1' } as SsdpDevice)).toBe('MiniUPnPd')
  })
})
