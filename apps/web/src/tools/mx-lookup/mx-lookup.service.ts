import type { components } from '@/api/schema'

export type EmailMx = components['schemas']['EmailMx']
export type Probe = EmailMx['hosts'][number]['probes'][number]

export function probeStatus(p: Probe): 'ok' | 'no-starttls' | 'cert' | 'unreachable' {
  if (!p.connected)
    return 'unreachable'
  if (!p.starttls)
    return 'no-starttls'
  // STARTTLS offered: a failed handshake or an invalid certificate is a TLS problem.
  return p.tls?.verified ? 'ok' : 'cert'
}
