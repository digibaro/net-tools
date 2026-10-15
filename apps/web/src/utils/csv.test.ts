import { describe, expect, it } from 'vitest'
import { csvName, toCsv } from './csv'

describe('toCsv', () => {
  it('quotes only where RFC 4180 requires it', () => {
    expect(toCsv(['Name', 'Value'], [['a', 1], ['b, c', 'say "hi"'], ['multi\nline', null]]))
      .toBe(`${String.fromCharCode(0xFEFF)}Name,Value\r\na,1\r\n"b, c","say ""hi"""\r\n"multi\nline",\r\n`)
  })
  it('names the file after the JSON download', () => {
    expect(csvName('port-scanner.json')).toBe('port-scanner.csv')
    expect(csvName('result')).toBe('result.csv')
  })
})
