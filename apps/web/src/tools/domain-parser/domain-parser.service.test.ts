import { describe, expect, it } from 'vitest'
import { analyzeDomain } from './domain-parser.service'

describe('analyzeDomain', () => {
  it('splits a multi-label ICANN suffix', () => {
    expect(analyzeDomain('www.shop.example.co.uk')?.withPrivate).toEqual({
      hostname: 'www.shop.example.co.uk',
      subdomain: 'www.shop',
      domain: 'example.co.uk',
      domainWithoutSuffix: 'example',
      publicSuffix: 'co.uk',
      suffixType: 'icann',
    })
  })

  it('distinguishes private suffixes', () => {
    const r = analyzeDomain('https://user.github.io/repo?x=1')!
    expect(r.withPrivate).toMatchObject({ domain: 'user.github.io', publicSuffix: 'github.io', subdomain: '', suffixType: 'private' })
    expect(r.icannOnly).toMatchObject({ domain: 'github.io', publicSuffix: 'io', subdomain: 'user', suffixType: 'icann' })
  })

  it('handles IPs and invalid input', () => {
    expect(analyzeDomain('192.0.2.1')?.isIp).toBe(true)
    expect(analyzeDomain('')).toBeUndefined()
    expect(analyzeDomain('bad_host!.com')).toBeUndefined()
  })
})
