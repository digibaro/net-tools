import { describe, expect, it } from 'vitest'
import { isLocallyAdministered, isMulticast } from '@/utils/mac'
import { formatAs, formatList, generateMacs, parsePrefix } from './mac-formatter.service'

const MAC = 0x001A2B3C4D5En

describe('formatAs', () => {
  it.each([
    ['colon', false, '00:1a:2b:3c:4d:5e'],
    ['hyphen', true, '00-1A-2B-3C-4D-5E'],
    ['cisco', false, '001a.2b3c.4d5e'],
    ['plain', true, '001A2B3C4D5E'],
  ] as const)('%s (upper=%s)', (f, upper, want) => {
    expect(formatAs(MAC, f, upper)).toBe(want)
  })
})

describe('formatList', () => {
  it('normalizes and reports invalid entries', () => {
    expect(formatList('001a.2b3c.4d5e, 00-1A-2B-3C-4D-5F bad', 'colon', false)).toEqual({ output: ['00:1a:2b:3c:4d:5e', '00:1a:2b:3c:4d:5f'], invalid: ['bad'] })
  })
})

describe('generateMacs', () => {
  it('makes locally administered unicast addresses by default', () => {
    const macs = generateMacs(50, { value: 0n, bits: 0 })
    expect(macs).toHaveLength(50)
    for (const m of macs) {
      expect(isLocallyAdministered(m)).toBe(true)
      expect(isMulticast(m)).toBe(false)
    }
  })

  it('keeps a given prefix', () => {
    const [m] = generateMacs(1, parsePrefix('00:0c:29')!, () => new Uint8Array([0xFF, 0xFF, 0xFF, 0xAA, 0xBB, 0xCC]))
    expect(formatAs(m!, 'colon')).toBe('00:0c:29:aa:bb:cc')
  })

  it('parsePrefix', () => {
    expect(parsePrefix('')).toEqual({ value: 0n, bits: 0 })
    expect(parsePrefix('00-0C-29')).toEqual({ value: 0x000C29n, bits: 24 })
    expect(parsePrefix('0')).toBeUndefined()
    expect(parsePrefix('00:11:22:33:44:55')).toBeUndefined()
  })
})
