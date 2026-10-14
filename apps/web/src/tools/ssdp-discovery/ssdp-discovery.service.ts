import type { components } from '@/api/schema'

export type SsdpDevice = components['schemas']['SsdpDevice']

/** "urn:schemas-upnp-org:device:InternetGatewayDevice:1" → "InternetGatewayDevice"; "upnp:rootdevice" stays. */
export function shortUrn(urn: string): string {
  const m = /^urn:[^:]+:(?:device|service):([^:]+)(?::\d+)?$/.exec(urn)
  return m ? m[1]! : urn
}

/** One-line label for a device: friendly name, else model, else the server header. */
export function label(d: SsdpDevice): string {
  return d.friendlyName || d.model || d.server || d.address
}
