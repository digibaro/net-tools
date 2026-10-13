import { describe, expect, it } from 'vitest'
import { clientHints, parseUserAgent } from './user-agent-parser.service'

describe('parseUserAgent', () => {
  it('parses desktop Firefox', () => {
    expect(parseUserAgent('Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0')).toEqual({
      browser: 'Firefox 128.0',
      engine: 'Gecko 128.0',
      os: 'Linux',
      device: '',
      deviceType: 'desktop',
      cpu: 'amd64',
    })
  })

  it('parses mobile Safari on iPhone', () => {
    const r = parseUserAgent('Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1')!
    expect(r).toMatchObject({ browser: 'Mobile Safari 17.5', os: 'iOS 17.5', device: 'Apple iPhone', deviceType: 'mobile' })
  })

  it('handles curl and empty input', () => {
    expect(parseUserAgent('curl/8.18.0')?.browser).toBe('')
    expect(parseUserAgent('  ')).toBeUndefined()
  })
})

describe('clientHints', () => {
  it('drops GREASE brands', () => {
    const nav = { userAgentData: { brands: [{ brand: 'Not)A;Brand', version: '99' }, { brand: 'Chromium', version: '156' }], platform: 'Linux', mobile: false } } as unknown as Navigator
    expect(clientHints(nav)).toEqual({ brands: 'Chromium 156', platform: 'Linux', mobile: false })
    expect(clientHints({} as Navigator)).toBeUndefined()
  })
})
