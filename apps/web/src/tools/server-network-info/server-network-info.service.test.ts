import { describe, expect, it } from 'vitest'
import { sortInterfaces } from './server-network-info.service'

const i = (name: string, flags: string[]) => ({ name, mtu: 1500, mac: '', flags, addresses: [] })

describe('sortInterfaces', () => {
  it('puts loopback and down interfaces last', () => {
    expect(sortInterfaces([i('lo', ['up', 'loopback']), i('eth1', []), i('eth0', ['up'])]).map(x => x.name)).toEqual(['eth0', 'eth1', 'lo'])
  })
})
