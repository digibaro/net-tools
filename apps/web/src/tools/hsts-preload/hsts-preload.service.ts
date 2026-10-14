import type { components } from '@/api/schema'

export type HstsPreload = components['schemas']['HstsPreload']

export function statusType(s: string): 'success' | 'warning' | 'error' | 'default' {
  switch (s) {
    case 'preloaded':
      return 'success'
    case 'pending':
      return 'warning'
    case 'rejected':
    case 'removed':
      return 'error'
  }
  return 'default'
}
