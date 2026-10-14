/** 48-bit MAC address helpers (shared by MAC, OUI, EUI-64 and ULA tools). */

/** Accepts aa:bb:cc:dd:ee:ff, aa-bb-…, aabb.ccdd.eeff (Cisco), aabbccddeeff; case-insensitive. */
export function parseMac(input: string): bigint | undefined {
  const s = input.trim().toLowerCase()
  const hex
    = /^[0-9a-f]{2}([:-])[0-9a-f]{2}(?:\1[0-9a-f]{2}){4}$/.test(s)
      ? s.replace(/[:-]/g, '')
      : /^[0-9a-f]{4}\.[0-9a-f]{4}\.[0-9a-f]{4}$/.test(s)
        ? s.replaceAll('.', '')
        : /^[0-9a-f]{12}$/.test(s)
          ? s
          : undefined
  return hex === undefined ? undefined : BigInt(`0x${hex}`)
}

export const macHex = (mac: bigint) => mac.toString(16).padStart(12, '0')

export function formatMac(mac: bigint, separator: ':' | '-' = ':'): string {
  return macHex(mac).match(/../g)!.join(separator)
}

/** First octet bit 0: group (multicast) address. */
export const isMulticast = (mac: bigint) => ((mac >> 40n) & 1n) === 1n
/** First octet bit 1: locally administered (U/L bit). */
export const isLocallyAdministered = (mac: bigint) => ((mac >> 40n) & 2n) === 2n

/** Modified EUI-64 (RFC 4291 App. A): insert FFFE in the middle and flip the U/L bit. */
export function macToEui64(mac: bigint): bigint {
  const upper = mac >> 24n
  const lower = mac & 0xFFFFFFn
  return ((upper << 40n) | (0xFFFEn << 24n) | lower) ^ (0x02n << 56n)
}

/** OUI (first 24 bits) as 6 uppercase hex digits. */
export const ouiOf = (mac: bigint) => macHex(mac).slice(0, 6).toUpperCase()
