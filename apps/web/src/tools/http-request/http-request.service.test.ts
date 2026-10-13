import { describe, expect, it } from 'vitest'
import { parseHeaderLines } from './http-request.service'

describe('parseHeaderLines', () => {
  it('parses header lines', () => {
    expect(parseHeaderLines('Accept: application/json\n\nAuthorization:  Bearer abc:def ')).toEqual({ headers: { Accept: 'application/json', Authorization: 'Bearer abc:def' } })
  })
  it.each(['no colon', ': empty name', 'Bad Name: x', 'Content-Length: 5'])('rejects %j', (s) => {
    expect(parseHeaderLines(s)).toEqual({ error: s })
  })
})
