/** UTF-8 byte order mark: spreadsheet programs then read accented text correctly. */
const BOM = String.fromCharCode(0xFEFF)

/** One CSV field (RFC 4180): quoted when it contains a comma, quote or line break. */
function field(value: unknown): string {
  const s = value === null || value === undefined ? '' : String(value)
  return /[",\r\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

/**
 * RFC 4180 CSV (CRLF line endings) with a UTF-8 byte order mark, so spreadsheet programs read
 * accented text correctly.
 */
export function toCsv(header: string[], rows: unknown[][]): string {
  return `${BOM}${[header, ...rows].map(r => r.map(field).join(',')).join('\r\n')}\r\n`
}

/** "port-scanner.json" → "port-scanner.csv" */
export const csvName = (filename: string) => `${filename.replace(/\.[a-z0-9]+$/i, '')}.csv`
