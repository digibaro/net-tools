import type { components } from '@/api/schema'

export type WellKnown = components['schemas']['WellKnown']
export type WellKnownFile = components['schemas']['WellKnownFile']

export function fileState(f: WellKnownFile): 'found' | 'missing' | 'error' {
  if (f.found)
    return 'found'
  return f.status === 404 || f.status === 410 ? 'missing' : 'error'
}
