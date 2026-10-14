import { describe, expect, it } from 'vitest'
import { escapeWifi, qrSvg, validate, wifiUri } from './wifi-qr-generator.service'

describe('wifiUri', () => {
  it('builds the WIFI: string', () => {
    expect(wifiUri({ ssid: 'Home', password: 'secret123', security: 'WPA', hidden: false })).toBe('WIFI:T:WPA;S:Home;P:secret123;;')
    expect(wifiUri({ ssid: 'Guest', password: 'ignored', security: 'nopass', hidden: true })).toBe('WIFI:T:nopass;S:Guest;H:true;;')
  })

  it('escapes reserved characters', () => {
    expect(escapeWifi(String.raw`a;b,c"d:e\f`)).toBe(String.raw`a\;b\,c\"d\:e\\f`)
    expect(wifiUri({ ssid: 'Café;1', password: 'p:ss', security: 'SAE', hidden: false })).toBe(String.raw`WIFI:T:SAE;S:Café\;1;P:p\:ss;;`)
  })
})

describe('validate', () => {
  it.each([
    [{ ssid: '', password: 'x', security: 'nopass' }, 'ssid'],
    [{ ssid: 'x'.repeat(33), password: '', security: 'nopass' }, 'ssid'],
    [{ ssid: 'Home', password: 'short', security: 'WPA' }, 'password'],
    [{ ssid: 'Home', password: '12345', security: 'WEP' }, undefined],
    [{ ssid: 'Home', password: '12345678', security: 'SAE' }, undefined],
    [{ ssid: 'Open', password: '', security: 'nopass' }, undefined],
  ] as const)('%j → %s', (cfg, want) => {
    expect(validate({ ...cfg, hidden: false })).toBe(want)
  })
})

describe('qrSvg', () => {
  it('renders an SVG', async () => {
    expect(await qrSvg('WIFI:T:WPA;S:x;P:12345678;;')).toMatch(/^<svg[\s\S]+<\/svg>\s*$/)
  })
})
