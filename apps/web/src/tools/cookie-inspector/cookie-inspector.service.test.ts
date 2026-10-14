import type { CookieInfo } from './cookie-inspector.service'
import { describe, expect, it } from 'vitest'
import { attributes, severity } from './cookie-inspector.service'

const c = (x: Partial<CookieInfo>) => ({ secure: false, httpOnly: false, sameSite: '', partitioned: false, domain: '', path: '', problems: [], warnings: [], ...x }) as CookieInfo

describe('cookie-inspector', () => {
  it('summarizes attributes', () => {
    expect(attributes(c({ secure: true, httpOnly: true, sameSite: 'Lax', path: '/' }))).toBe('Secure · HttpOnly · SameSite=Lax · Path=/')
  })
  it('rates cookies', () => {
    expect(severity(c({ problems: ['host-prefix'] }))).toBe('error')
    expect(severity(c({ warnings: ['not-secure'] }))).toBe('warning')
    expect(severity(c({ warnings: ['deleted'] }))).toBe('success')
  })
})
