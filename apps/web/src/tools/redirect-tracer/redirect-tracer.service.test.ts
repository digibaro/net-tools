import type { HttpHop } from './redirect-tracer.service'
import { describe, expect, it } from 'vitest'
import { summarize } from './redirect-tracer.service'

const h = (url: string, status: number, errorCode = ''): HttpHop => ({ url, status, errorCode } as HttpHop)

describe('summarize', () => {
  it('detects upgrades and downgrades', () => {
    expect(summarize([h('http://a', 301), h('https://a', 302), h('https://b', 200)])).toMatchObject({ redirects: 2, httpsUpgrade: true, downgrade: false, finalStatus: 200 })
    expect(summarize([h('https://a', 302), h('http://b', 200)]).downgrade).toBe(true)
    expect(summarize([h('https://a', 302), h('http://10.0.0.1', 0, 'TARGET_PRIVATE_DENIED')]).stoppedBy).toBe('TARGET_PRIVATE_DENIED')
  })
})
