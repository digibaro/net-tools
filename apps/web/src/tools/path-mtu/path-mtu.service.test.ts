import { describe, expect, it } from 'vitest'
import { likelyCause } from './path-mtu.service'

describe('likelyCause', () => {
  it('names common MTUs', () => {
    expect(likelyCause(1492)).toBe('pppoe')
    expect(likelyCause(1420)).toBe('wireguard')
    expect(likelyCause(1337)).toBeUndefined()
  })
})
