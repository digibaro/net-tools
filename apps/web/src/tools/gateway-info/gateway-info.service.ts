import type { components } from '@/api/schema'
import type { KeyValueRow } from '@/ui/c-key-value-list.vue'

export type GatewayResult = components['schemas']['GatewayResult']

type T = (key: string, params?: Record<string, unknown>) => string

/** Key/value rows for the result; `t` resolves keys under tools.gateway-info. */
export function gatewayRows(r: GatewayResult, t: T): KeyValueRow[] {
  const dash = '—'
  const pmp = r.natPmp.supported ? r.natPmp.externalIp : (r.natPmp.error || t('no'))
  const pcp = r.natPmp.pcp ? t('yes') : t('no')
  const upnp = r.upnp.supported
    ? [r.upnp.externalIp || r.upnp.error, [r.upnp.friendlyName, r.upnp.model].filter(Boolean).join(' · ')].filter(Boolean).join(' — ')
    : (r.upnp.error || t('no'))
  return [
    { label: t('rows.interface'), value: r.interface, mono: true },
    { label: t('rows.addresses'), value: r.addresses.join(', ') || dash, mono: true },
    { label: t('rows.gateway'), value: r.gateway, mono: true },
    { label: t('rows.gatewayMac'), value: r.gatewayMac || dash, mono: true },
    { label: t('rows.externalIp'), value: r.externalIp || dash, mono: true },
    { label: 'NAT-PMP', value: pmp, noCopy: !r.natPmp.supported },
    { label: 'PCP', value: pcp, noCopy: true },
    { label: 'UPnP IGD', value: upnp, noCopy: !r.upnp.supported },
  ]
}
