import type { components } from '@/api/schema'

export type RpkiResult = components['schemas']['RpkiResult']
export { isPrefixOrIp, parseAsn } from '@/utils/bgp'

export function statusType(status: string): 'success' | 'error' | 'default' {
  if (status === 'valid')
    return 'success'
  return status.startsWith('invalid') ? 'error' : 'default'
}
