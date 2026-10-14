import type { MdnsService } from './mdns-browser.service'
import { describe, expect, it } from 'vitest'
import { describeType, instanceName, shortType } from './mdns-browser.service'

describe('mdns-browser', () => {
  it('formats types and names', () => {
    expect(shortType('_ipp._tcp.local.')).toBe('_ipp._tcp')
    expect(describeType('_ipp._tcp.local.')).toBe('Printer (IPP)')
    expect(instanceName({ instance: 'Office\\ Printer\\0402._ipp._tcp.local.', type: '_ipp._tcp.local.' } as MdnsService)).toBe('Office Printer(2')
  })
})
