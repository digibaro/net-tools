import { describe, expect, it } from 'vitest'
import { STATUS_CODES } from './http-status-codes.data'
import { filterCodes, rfcUrl } from './http-status-codes.service'

describe('http status codes', () => {
  it('data is sorted and unique', () => {
    const codes = STATUS_CODES.map(c => c.code)
    expect(codes).toEqual([...new Set(codes)].sort((a, b) => a - b))
  })

  it('filters by code prefix, text and class', () => {
    expect(filterCodes('42', 'all').map(c => c.code)).toEqual([421, 422, 423, 424, 425, 426, 428, 429])
    expect(filterCodes('teapot', 'all').map(c => c.code)).toEqual([418])
    expect(filterCodes('redirect', '3').map(c => c.code)).toEqual([301, 302, 303, 307, 308])
    expect(filterCodes('', '1')).toHaveLength(4)
  })

  it('links to the RFC section', () => {
    expect(rfcUrl({ rfc: 9110, section: '15.5.5' })).toBe('https://www.rfc-editor.org/rfc/rfc9110#section-15.5.5')
  })
})
