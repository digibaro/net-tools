import { describe, expect, it } from 'vitest'
import { convertDomain, convertList } from './punycode-converter.service'

describe('convertDomain', () => {
  it.each([
    ['münchen.de', 'xn--mnchen-3ya.de', 'münchen.de', 1],
    ['xn--mnchen-3ya.de', 'xn--mnchen-3ya.de', 'münchen.de', 1],
    ['MÜNCHEN.DE', 'xn--mnchen-3ya.de', 'münchen.de', 1],
    ['例え.テスト', 'xn--r8jz45g.xn--zckzah', '例え.テスト', 2],
    ['example.com.', 'example.com', 'example.com', 0],
    ['bücher.example', 'xn--bcher-kva.example', 'bücher.example', 1],
  ])('%s', (input, ascii, unicode, idn) => {
    expect(convertDomain(input)).toMatchObject({ ascii, unicode, idnLabels: idn })
  })

  it.each(['', 'a b.com', 'http://x.com', 'user@x.com', 'x.com/path'])('rejects %j', (s) => {
    expect(convertDomain(s)).toBeUndefined()
  })
})

describe('convertList', () => {
  it('splits by line and comma', () => {
    const r = convertList('münchen.de, xn--bcher-kva.example\nbad domain')
    expect(r.rows.map(x => x.unicode)).toEqual(['münchen.de', 'bücher.example'])
    expect(r.invalid).toEqual(['bad domain'])
  })
})
