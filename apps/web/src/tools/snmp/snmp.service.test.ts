import { describe, expect, it } from 'vitest'
import { displayValue, formatTicks, isOidList, oidName, splitOids } from './snmp.service'

describe('snmp', () => {
  it('names OIDs', () => {
    expect(oidName('.1.3.6.1.2.1.1.1.0')).toBe('sysDescr.0')
    expect(oidName('.1.3.6.1.2.1.2.2.1.2.3')).toBe('ifDescr.3')
    expect(oidName('1.3.6.1.2.1.2.2')).toBe('ifTable')
    expect(oidName('.1.3.6.1.4.1.9.9.1')).toBe('.1.3.6.1.4.1.9.9.1')
  })
  it('formats values', () => {
    expect(formatTicks(9006100)).toBe('1d 1h 1m 1s')
    expect(displayValue({ oid: '', type: 'Hex-STRING', value: '001122AABBCC' })).toBe('00:11:22:aa:bb:cc')
    expect(displayValue({ oid: '', type: 'TimeTicks', value: '6000' })).toBe('1m 0s (6000)')
  })
  it('parses OID lists', () => {
    expect(isOidList('1.3.6.1.2.1.1.1.0, .1.3.6.1.2.1.1.5.0')).toBe(true)
    expect(isOidList('sysDescr')).toBe(false)
    expect(splitOids('1.3.6 1.3.7,1.3.8')).toEqual(['1.3.6', '1.3.7', '1.3.8'])
  })
})
