import { describe, expect, it } from 'vitest'
import { lookup, parseMacOrOui, parseOuiTable } from './mac-address-lookup.service'

const table = parseOuiTable('000C29\tVMware, Inc.\tUnited States\n0050C2\tIEEE Registration Authority\t\n')

describe('mac-address-lookup', () => {
  it('parses the table', () => {
    expect(table.get('000C29')).toEqual({ vendor: 'VMware, Inc.', country: 'United States' })
    expect(table.get('0050C2')).toEqual({ vendor: 'IEEE Registration Authority', country: '' })
  })

  it('looks up a full MAC', () => {
    expect(lookup('00-0c-29-ab-cd-ef', table)).toEqual({
      input: '00-0c-29-ab-cd-ef',
      mac: '00:0c:29:ab:cd:ef',
      oui: '00:0C:29',
      vendor: 'VMware, Inc.',
      country: 'United States',
      administration: 'universal',
      cast: 'unicast',
    })
  })

  it('accepts a bare OUI and flags special bits', () => {
    expect(lookup('000c29', table)).toMatchObject({ mac: '', vendor: 'VMware, Inc.' })
    expect(lookup('02:00:00:00:00:01', table)).toMatchObject({ vendor: '', administration: 'local' })
    expect(lookup('01:00:5e:00:00:fb', table)).toMatchObject({ cast: 'multicast' })
    expect(lookup('ff:ff:ff:ff:ff:ff', table)).toMatchObject({ cast: 'broadcast' })
  })

  it.each(['00:0c:29:ab', 'xyz', ''])('rejects %j', (s) => {
    expect(parseMacOrOui(s)).toBeUndefined()
  })
})
