import { toAscii } from '@/tools/punycode-converter/punycode-converter.service'
import { parseIP } from './ip'

/**
 * A dial target as the API accepts it: hostname (IDN ok) or IPv4/IPv6 literal. The server
 * applies the full policy; this only catches obvious typos early.
 */
export function isValidHost(input: string): boolean {
  const s = input.trim().replace(/^\[(.*)\]$/, '$1')
  if (!s || /[\s/@?#]/.test(s))
    return false
  if (parseIP(s))
    return true
  if (/^[\d.]+$/.test(s) || /^0x/i.test(s))
    return false // ambiguous numeric forms are rejected by the server too
  const ascii = toAscii(s)
  return !!ascii && /^[\w-]+(?:\.[\w-]+)*$/.test(ascii)
}
