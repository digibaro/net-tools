import type { EchoInfo } from '@/api/client'

export type HintId = 'via' | 'forwarded' | 'multi-hop' | 'cdn' | 'compression-stripped'

/** Headers set by NetTools' own nginx in front of the API; ignored for proxy detection. */
const OWN_PROXY = new Set(['x-real-ip', 'x-forwarded-proto'])
const CDN_HEADERS = ['cf-connecting-ip', 'cf-ray', 'true-client-ip', 'fastly-client-ip', 'x-akamai-edgescape', 'x-azure-ref', 'x-amz-cf-id']

/** Signs that something other than our own nginx (a proxy, CDN or middlebox) sat in the path. */
export function proxyHints(echo: Pick<EchoInfo, 'headers'>): HintId[] {
  const h = new Map(echo.headers.map(x => [x.name.toLowerCase(), x.value]))
  const hints: HintId[] = []
  if (h.has('via'))
    hints.push('via')
  if (h.has('forwarded'))
    hints.push('forwarded')
  // Our nginx appends exactly one hop; more than one means another proxy added entries.
  const xff = h.get('x-forwarded-for')
  if (xff && xff.split(',').length > 1)
    hints.push('multi-hop')
  if (CDN_HEADERS.some(c => h.has(c)))
    hints.push('cdn')
  if (h.has('user-agent') && !h.has('accept-encoding'))
    hints.push('compression-stripped')
  return hints
}

export function headerRows(echo: Pick<EchoInfo, 'headers'>): { name: string, value: string, addedByNetTools: boolean }[] {
  return echo.headers.map(x => ({
    name: x.name,
    value: x.value,
    addedByNetTools: OWN_PROXY.has(x.name.toLowerCase()) || x.name.toLowerCase() === 'x-forwarded-for',
  }))
}
