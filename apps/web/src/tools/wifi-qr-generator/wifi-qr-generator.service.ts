import QRCode from 'qrcode'

export type WifiSecurity = 'WPA' | 'SAE' | 'WEP' | 'nopass'
export const SECURITIES: WifiSecurity[] = ['WPA', 'SAE', 'WEP', 'nopass']

/** Backslash-escape the characters the WIFI: URI format reserves. */
export function escapeWifi(s: string): string {
  return s.replace(/[\\;,":]/g, c => `\\${c}`)
}

export interface WifiConfig {
  ssid: string
  password: string
  security: WifiSecurity
  hidden: boolean
}

/** WIFI:T:<type>;S:<ssid>;P:<password>;H:true;; (ZXing / Wi-Fi Alliance format; SAE = WPA3). */
export function wifiUri({ ssid, password, security, hidden }: WifiConfig): string {
  const parts = [`T:${security}`, `S:${escapeWifi(ssid)}`]
  if (security !== 'nopass')
    parts.push(`P:${escapeWifi(password)}`)
  if (hidden)
    parts.push('H:true')
  return `WIFI:${parts.join(';')};;`
}

export function validate(cfg: WifiConfig): 'ssid' | 'password' | undefined {
  if (!cfg.ssid || new TextEncoder().encode(cfg.ssid).length > 32)
    return 'ssid'
  if (cfg.security === 'WPA' || cfg.security === 'SAE')
    return cfg.password.length >= 8 && cfg.password.length <= 63 ? undefined : 'password'
  if (cfg.security === 'WEP')
    return [5, 13, 10, 26].includes(cfg.password.length) ? undefined : 'password'
  return undefined
}

// QR codes must stay dark-on-light in both themes so phone cameras can scan them.
const QR_COLORS = { dark: '#000000', light: '#ffffff' }

export function qrSvg(text: string): Promise<string> {
  return QRCode.toString(text, { type: 'svg', errorCorrectionLevel: 'M', margin: 2, color: QR_COLORS })
}

export function qrPngDataUrl(text: string, width = 512): Promise<string> {
  return QRCode.toDataURL(text, { errorCorrectionLevel: 'M', margin: 2, width, color: QR_COLORS })
}
