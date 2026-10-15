import type { components } from '@/api/schema'

export type MixedResult = components['schemas']['MixedResult']

export type Verdict = 'clean' | 'upgraded' | 'passive' | 'active'

/** active: browsers block it (page breaks); passive: shown with a warning or auto-upgraded. */
export function verdict(r: MixedResult): Verdict {
  if (r.active + r.passive + r.forms === 0)
    return 'clean'
  if (r.upgradeInsecureRequests)
    return 'upgraded'
  return r.active + r.forms > 0 ? 'active' : 'passive'
}
