import type { components } from '@/api/schema'

export type EmailDkim = components['schemas']['EmailDkim']

/** Selectors commonly used by large providers, offered as suggestions. */
export const COMMON_SELECTORS = ['default', 'google', 'selector1', 'selector2', 'k1', 's1', 's2', 'dkim', 'mail', 'mxvault']

export const isValidSelector = (s: string) => /^[\w-]+(?:\.[\w-]+)*$/.test(s.trim())
