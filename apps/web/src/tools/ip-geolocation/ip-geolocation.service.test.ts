import { describe, expect, it } from 'vitest'
import { flag, mapUrl } from './ip-geolocation.service'

describe('ip-geolocation helpers', () => {
  it('builds map links', () => {
    expect(mapUrl(52.37, 4.89)).toBe('https://www.openstreetmap.org/?mlat=52.37&mlon=4.89#map=10/52.37/4.89')
  })
  it('makes flags', () => {
    expect(flag('nl')).toBe('🇳🇱')
    expect(flag('')).toBe('')
    expect(flag('EU1')).toBe('')
  })
})
