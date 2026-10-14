import type { components } from '@/api/schema'

export type ScanProbe = components['schemas']['TlsScanProbe']
export type ScanSummary = components['schemas']['TlsScanSummary']

/** Supported cipher probes grouped by version, newest version first. */
export function ciphersByVersion(probes: ScanProbe[]): { version: string, ciphers: ScanProbe[] }[] {
  const map = new Map<string, ScanProbe[]>()
  for (const p of probes) {
    if (p.kind === 'cipher' && p.supported)
      map.set(p.version, [...(map.get(p.version) ?? []), p])
  }
  return [...map].map(([version, ciphers]) => ({ version, ciphers: ciphers.sort((a, b) => a.weak.length - b.weak.length || a.name.localeCompare(b.name)) })).sort((a, b) => b.version.localeCompare(a.version))
}
