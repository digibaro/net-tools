import { describe, expect, it } from 'vitest'
import { countPorts, isValidPorts } from './port-check.service'

describe('ports', () => {
  it('counts unique ports', () => {
    expect(countPorts('22, 80,443,80')).toBe(3)
    expect(countPorts('8000-8004')).toBe(5)
  })
  it.each(['', 'x', '0', '70000', '1-20', '1,2,3,4,5,6,7,8,9,10,11'])('rejects %j', (s) => {
    expect(isValidPorts(s)).toBe(false)
  })
})
