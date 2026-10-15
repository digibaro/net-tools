import type { LldpNeighbor } from './lldp-neighbors.service'
import { describe, expect, it } from 'vitest'
import { deviceName, isInterfaceName } from './lldp-neighbors.service'

describe('lldp-neighbors', () => {
  it('validates interface names', () => {
    expect(isInterfaceName('')).toBe(true)
    expect(isInterfaceName('enp3s0')).toBe(true)
    expect(isInterfaceName('a b')).toBe(false)
  })
  it('names the device', () => {
    expect(deviceName({ systemName: '', chassisId: '00:11:22:33:44:00', sourceMac: 'x' } as LldpNeighbor)).toBe('00:11:22:33:44:00')
  })
})
