import { describe, expect, it } from 'vitest'
import { computeGlobalId, ntpTimestamp, parseGlobalId, randomEui64, ulaFromGlobalId } from './ipv6-ula-generator.service'

describe('ntpTimestamp', () => {
  it('converts the Unix epoch', () => {
    expect(ntpTimestamp(0)).toBe(2_208_988_800n << 32n)
    expect(ntpTimestamp(500) & 0xFFFFFFFFn).toBe(1n << 31n)
  })
})

describe('computeGlobalId', () => {
  it('is the low 40 bits of SHA-1(time ‖ EUI-64)', async () => {
    const ntp = 0x0102030405060708n
    const eui = 0x1122334455667788n
    // python3: hashlib.sha1(bytes.fromhex('0102…7788')).hexdigest()[-10:]
    const ref = '8f36fec473'
    expect((await computeGlobalId(ntp, eui)).toString(16).padStart(10, '0')).toBe(ref)
  })
})

describe('ulaFromGlobalId', () => {
  it('builds the fd00::/8 prefix and subnets', () => {
    expect(ulaFromGlobalId(0x12_3456_789An)).toMatchObject({
      globalId: '123456789a',
      prefix: 'fd12:3456:789a::/48',
      firstSubnet: 'fd12:3456:789a::/64',
      lastSubnet: 'fd12:3456:789a:ffff::/64',
      exampleSubnets: ['fd12:3456:789a:1::/64', 'fd12:3456:789a:2::/64', 'fd12:3456:789a:3::/64', 'fd12:3456:789a:10::/64'],
    })
    expect(ulaFromGlobalId(0n).prefix).toBe('fd00::/48')
  })
})

describe('helpers', () => {
  it('parseGlobalId accepts exactly 10 hex digits', () => {
    expect(parseGlobalId('123456789A')).toBe(0x123456789An)
    expect(parseGlobalId('123')).toBeUndefined()
    expect(parseGlobalId('123456789g')).toBeUndefined()
  })
  it('randomEui64 is 64-bit', () => {
    expect(randomEui64() < 1n << 64n).toBe(true)
  })
})
