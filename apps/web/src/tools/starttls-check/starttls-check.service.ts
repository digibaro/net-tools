import { DEFAULT_PORTS, STARTTLS } from '../tls-inspector/tls-inspector.service'

/** Protocols offered by the STARTTLS check, with their usual plaintext ports. */
export const PROTOCOLS = STARTTLS.filter(Boolean).map(p => ({ id: p, port: DEFAULT_PORTS[p] }))
