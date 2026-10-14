import type { components } from '@/api/schema'

export type SnmpRequest = components['schemas']['SnmpRequest']
export type SnmpResult = components['schemas']['SnmpResult']
export type SnmpVar = SnmpResult['vars'][number]

/** Common MIB-2 / IF-MIB / HOST-RESOURCES objects (prefix → name). */
const NAMES: Record<string, string> = {
  '1.3.6.1.2.1.1': 'system',
  '1.3.6.1.2.1.1.1': 'sysDescr',
  '1.3.6.1.2.1.1.2': 'sysObjectID',
  '1.3.6.1.2.1.1.3': 'sysUpTime',
  '1.3.6.1.2.1.1.4': 'sysContact',
  '1.3.6.1.2.1.1.5': 'sysName',
  '1.3.6.1.2.1.1.6': 'sysLocation',
  '1.3.6.1.2.1.1.7': 'sysServices',
  '1.3.6.1.2.1.2.1': 'ifNumber',
  '1.3.6.1.2.1.2.2': 'ifTable',
  '1.3.6.1.2.1.2.2.1.1': 'ifIndex',
  '1.3.6.1.2.1.2.2.1.2': 'ifDescr',
  '1.3.6.1.2.1.2.2.1.3': 'ifType',
  '1.3.6.1.2.1.2.2.1.4': 'ifMtu',
  '1.3.6.1.2.1.2.2.1.5': 'ifSpeed',
  '1.3.6.1.2.1.2.2.1.6': 'ifPhysAddress',
  '1.3.6.1.2.1.2.2.1.7': 'ifAdminStatus',
  '1.3.6.1.2.1.2.2.1.8': 'ifOperStatus',
  '1.3.6.1.2.1.2.2.1.9': 'ifLastChange',
  '1.3.6.1.2.1.2.2.1.10': 'ifInOctets',
  '1.3.6.1.2.1.2.2.1.13': 'ifInDiscards',
  '1.3.6.1.2.1.2.2.1.14': 'ifInErrors',
  '1.3.6.1.2.1.2.2.1.16': 'ifOutOctets',
  '1.3.6.1.2.1.2.2.1.19': 'ifOutDiscards',
  '1.3.6.1.2.1.2.2.1.20': 'ifOutErrors',
  '1.3.6.1.2.1.4.20': 'ipAddrTable',
  '1.3.6.1.2.1.4.20.1.1': 'ipAdEntAddr',
  '1.3.6.1.2.1.4.20.1.2': 'ipAdEntIfIndex',
  '1.3.6.1.2.1.4.20.1.3': 'ipAdEntNetMask',
  '1.3.6.1.2.1.25.1.1': 'hrSystemUptime',
  '1.3.6.1.2.1.25.2.3': 'hrStorageTable',
  '1.3.6.1.2.1.25.3.3.1.2': 'hrProcessorLoad',
  '1.3.6.1.2.1.31.1.1.1.1': 'ifName',
  '1.3.6.1.2.1.31.1.1.1.6': 'ifHCInOctets',
  '1.3.6.1.2.1.31.1.1.1.10': 'ifHCOutOctets',
  '1.3.6.1.2.1.31.1.1.1.15': 'ifHighSpeed',
  '1.3.6.1.2.1.31.1.1.1.18': 'ifAlias',
  '1.3.6.1.2.1.47.1.1.1.1.2': 'entPhysicalDescr',
  '1.3.6.1.2.1.47.1.1.1.1.11': 'entPhysicalSerialNum',
}

/** Presets for the OID field (subtree to walk or a single instance). */
export const PRESETS = [
  { key: 'system', oid: '1.3.6.1.2.1.1' },
  { key: 'interfaces', oid: '1.3.6.1.2.1.2.2' },
  { key: 'ifx', oid: '1.3.6.1.2.1.31.1.1' },
  { key: 'ip', oid: '1.3.6.1.2.1.4.20' },
  { key: 'storage', oid: '1.3.6.1.2.1.25.2.3' },
] as const

/** ".1.3.6.1.2.1.2.2.1.2.3" → "ifDescr.3" (longest known prefix), unknown OIDs unchanged. */
export function oidName(oid: string): string {
  const parts = oid.replace(/^\./, '').split('.')
  for (let n = parts.length; n > 0; n--) {
    const name = NAMES[parts.slice(0, n).join('.')]
    if (name)
      return n === parts.length ? name : `${name}.${parts.slice(n).join('.')}`
  }
  return oid
}

/** TimeTicks (1/100 s) → "1d 2h 3m 4s". */
export function formatTicks(ticks: number): string {
  let s = Math.floor(ticks / 100)
  const d = Math.floor(s / 86400)
  s %= 86400
  const h = Math.floor(s / 3600)
  s %= 3600
  const m = Math.floor(s / 60)
  s %= 60
  return [d ? `${d}d` : '', h ? `${h}h` : '', m ? `${m}m` : '', `${s}s`].filter(Boolean).join(' ')
}

/** Human value: TimeTicks as a duration, 6-byte hex strings as MACs. */
export function displayValue(v: SnmpVar): string {
  if (v.type === 'TimeTicks')
    return `${formatTicks(Number(v.value))} (${v.value})`
  if (v.type === 'Hex-STRING' && v.value.length === 12)
    return v.value.match(/../g)!.join(':').toLowerCase()
  return v.value
}

export const isOidList = (s: string) => s.split(/[\s,]+/).filter(Boolean).every(o => /^\.?[0-2](?:\.\d{1,10})+$/.test(o))
export const splitOids = (s: string) => s.split(/[\s,]+/).filter(Boolean)
