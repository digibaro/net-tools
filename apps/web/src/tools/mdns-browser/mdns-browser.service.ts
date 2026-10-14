import type { components } from '@/api/schema'

export type MdnsService = components['schemas']['MdnsService']

const KNOWN: Record<string, string> = {
  '_http._tcp': 'Web server',
  '_https._tcp': 'Web server (TLS)',
  '_ipp._tcp': 'Printer (IPP)',
  '_ipps._tcp': 'Printer (IPPS)',
  '_printer._tcp': 'Printer (LPD)',
  '_pdl-datastream._tcp': 'Printer (raw)',
  '_smb._tcp': 'File sharing (SMB)',
  '_afpovertcp._tcp': 'File sharing (AFP)',
  '_ssh._tcp': 'SSH',
  '_sftp-ssh._tcp': 'SFTP',
  '_airplay._tcp': 'AirPlay',
  '_raop._tcp': 'AirPlay audio',
  '_googlecast._tcp': 'Chromecast',
  '_spotify-connect._tcp': 'Spotify Connect',
  '_homekit._tcp': 'HomeKit',
  '_hap._tcp': 'HomeKit accessory',
  '_matter._tcp': 'Matter',
  '_companion-link._tcp': 'Apple device',
  '_device-info._tcp': 'Device info',
  '_workstation._tcp': 'Workstation',
  '_scanner._tcp': 'Scanner',
  '_uscan._tcp': 'Scanner (eSCL)',
  '_sonos._tcp': 'Sonos',
  '_home-assistant._tcp': 'Home Assistant',
  '_mqtt._tcp': 'MQTT broker',
}

/** "_ipp._tcp.local." → "_ipp._tcp" */
export const shortType = (t: string) => t.replace(/\.local\.?$/, '')

export const describeType = (t: string) => KNOWN[shortType(t)] ?? ''

/** "Office\ Printer._ipp._tcp.local." → "Office Printer" */
export function instanceName(s: MdnsService): string {
  const suffix = `.${s.type}`
  const name = s.instance.endsWith(suffix) ? s.instance.slice(0, -suffix.length) : s.instance
  return name.replace(/\\(\d{3})/g, (_, d: string) => String.fromCharCode(Number(d))).replace(/\\(.)/g, '$1')
}
