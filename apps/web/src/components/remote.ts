import type { RemoteProbeResult } from '@/composables/useRemote'

type R = RemoteProbeResult['result']
export interface HopRow { hop: number, host: string, ip: string, asn: string, avg: string, loss: string }

const ms = (n: number | null | undefined) => (n === null || n === undefined ? '—' : `${Math.round(n * 10) / 10} ms`)

export function hopRows(r: R): HopRow[] {
  return (r.hops ?? []).map((h, i) => {
    const rtts = (h.timings ?? []).map(t => t.rtt).filter((v): v is number => typeof v === 'number')
    const avg = h.stats?.avg ?? (rtts.length ? rtts.reduce((a, b) => a + b, 0) / rtts.length : null)
    return {
      hop: i + 1,
      host: h.resolvedHostname && h.resolvedHostname !== h.resolvedAddress ? h.resolvedHostname : '',
      ip: h.resolvedAddress ?? '*',
      asn: (h.asn ?? []).map(a => `AS${a}`).join(' '),
      avg: ms(avg),
      loss: h.stats ? `${h.stats.loss} %` : '',
    }
  })
}

export function pingSummary(r: R): string {
  const s = r.stats
  if (!s || s.avg === null || s.avg === undefined)
    return s ? `${s.loss} % loss` : ''
  return `${ms(s.min)} / ${ms(s.avg)} / ${ms(s.max)} · ${s.loss} % loss`
}

/** "up" when the probe got an answer: ping replies, a final hop, DNS NOERROR, HTTP status. */
export function verdict(type: string, r: R): 'up' | 'down' | 'error' {
  if (r.status !== 'finished')
    return 'error'
  switch (type) {
    case 'ping':
      return (r.stats?.rcv ?? 0) > 0 ? 'up' : 'down'
    case 'http':
      return typeof r.statusCode === 'number' ? 'up' : 'down'
    case 'dns':
      return r.statusCodeName === 'NOERROR' ? 'up' : 'down'
    default: {
      const last = r.hops?.at(-1)
      return last?.resolvedAddress && last.resolvedAddress === r.resolvedAddress ? 'up' : 'down'
    }
  }
}

export function countryName(code: string, locale = 'en'): string {
  try {
    return new Intl.DisplayNames([locale], { type: 'region' }).of(code) ?? code
  }
  catch {
    return code
  }
}
