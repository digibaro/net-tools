import { createHash } from 'node:crypto'
import { describe, expect, it } from 'vitest'
import { inlineScriptHashes } from './csp-hashes'

describe('inlineScriptHashes', () => {
  it('hashes inline scripts only, byte-exact', () => {
    const body = '(function () { var x = 1 })()'
    const html = `<head><script>${body}</script><script type="module" src="/a.js"></script><script type="module">\nimport "x"\n</script></head>`
    const want = (s: string) => `'sha256-${createHash('sha256').update(s).digest('base64')}'`
    expect(inlineScriptHashes(html)).toEqual([want(body), want('\nimport "x"\n')])
  })

  it('returns nothing without inline scripts', () => {
    expect(inlineScriptHashes('<script src="/x.js"></script>')).toEqual([])
  })
})
