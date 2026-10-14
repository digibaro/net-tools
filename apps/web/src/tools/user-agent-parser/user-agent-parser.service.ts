import UAParser from 'ua-parser-js'

export interface UaInfo {
  browser: string
  engine: string
  os: string
  device: string
  deviceType: string
  cpu: string
}

const join = (...parts: (string | undefined)[]) => parts.filter(Boolean).join(' ')

export function parseUserAgent(ua: string): UaInfo | undefined {
  const s = ua.trim()
  if (!s)
    return undefined
  const r = new UAParser(s).getResult()
  return {
    browser: join(r.browser.name, r.browser.version),
    engine: join(r.engine.name, r.engine.version),
    os: join(r.os.name, r.os.version),
    device: join(r.device.vendor, r.device.model),
    deviceType: r.device.type ?? 'desktop',
    cpu: r.cpu.architecture ?? '',
  }
}

export interface ClientHints {
  brands: string
  platform: string
  mobile: boolean
}

interface NavigatorUAData {
  brands: { brand: string, version: string }[]
  platform: string
  mobile: boolean
}

/** User-Agent Client Hints (Chromium); undefined where unsupported. */
export function clientHints(nav: Navigator = navigator): ClientHints | undefined {
  const data = (nav as Navigator & { userAgentData?: NavigatorUAData }).userAgentData
  if (!data)
    return undefined
  return {
    brands: data.brands.filter(b => !/not.?a.?brand/i.test(b.brand)).map(b => `${b.brand} ${b.version}`).join(', '),
    platform: data.platform,
    mobile: data.mobile,
  }
}
