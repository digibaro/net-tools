import type { components } from '@/api/schema'

export type Geo = components['schemas']['Geo']
export { isIpAddress } from '../reverse-dns/reverse-dns.service'

/** OpenStreetMap link for the coordinates (opened in a new tab; nothing is embedded). */
export function mapUrl(lat: number, lon: number): string {
  const zoom = 10
  return `https://www.openstreetmap.org/?mlat=${lat}&mlon=${lon}#map=${zoom}/${lat}/${lon}`
}

/** Regional-indicator emoji flag for an ISO 3166-1 alpha-2 code. */
export function flag(code: string): string {
  if (!/^[A-Z]{2}$/i.test(code))
    return ''
  return String.fromCodePoint(...[...code.toUpperCase()].map(c => 0x1F1E6 + c.charCodeAt(0) - 65))
}
