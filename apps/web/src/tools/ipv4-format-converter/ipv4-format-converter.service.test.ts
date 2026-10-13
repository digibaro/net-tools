import { describe, expect, it } from 'vitest'
import { formats, parseAnyIPv4 } from './ipv4-format-converter.service'

const LOOPBACK = 0x7F000001n

describe('parseAnyIPv4', () => {
  it.each([
    ['127.0.0.1', 'dotted'],
    ['2130706433', 'inet_aton'],
    ['0x7f000001', 'inet_aton'],
    ['0x7F.1', 'inet_aton'],
    ['127.1', 'inet_aton'],
    ['0177.0.0.1', 'inet_aton'],
    ['127.0.1', 'inet_aton'],
    ['01111111.00000000.00000000.00000001', 'binary'],
    ['0b1111111000000000000000000000001', 'binary'],
  ])('%s → 127.0.0.1 (%s)', (s, form) => {
    expect(parseAnyIPv4(s)).toEqual({ value: LOOPBACK, form })
  })

  it.each(['256.0.0.1', '1.2.3.4.5', '0x100.0.0.1', '1.16777216', '4294967296', '08.0.0.1', 'abc', '', '1..2'])('rejects %j', (s) => {
    expect(parseAnyIPv4(s)).toBeUndefined()
  })
})

describe('formats', () => {
  it('renders all notations', () => {
    expect(formats(0xC0A8010An)).toEqual({
      dotted: '192.168.1.10',
      decimal: '3232235786',
      hex: '0xC0A8010A',
      hexDotted: '0xC0.0xA8.0x01.0x0A',
      octalDotted: '0300.0250.0001.0012',
      binaryDotted: '11000000.10101000.00000001.00001010',
      binary: '11000000101010000000000100001010',
      ipv4Mapped: '::ffff:192.168.1.10 (::ffff:c0a8:10a)',
    })
  })
})
