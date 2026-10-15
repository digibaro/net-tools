import { describe, expect, it } from 'vitest'
import { evaluate, newNonce, parseCsp, PRESETS, render } from './csp-builder.service'

const ids = (p: string) => evaluate(p).map(f => f.id)

describe('csp-builder', () => {
  it('parses directives', () => {
    expect(parseCsp(` default-src 'self' ; img-src * data: ;`)).toEqual([{ name: 'default-src', values: ['\'self\''] }, { name: 'img-src', values: ['*', 'data:'] }])
  })
  it('accepts the strict preset (fallbacks only)', () => {
    const f = evaluate(PRESETS.strict.replace('{RANDOM}', 'q83bdcDhlwX8d5nA8mBz3Q=='))
    expect(f.filter(x => x.severity !== 'info')).toEqual([])
    expect(f.map(x => x.id)).toContain('unsafeInlineFallback')
  })
  it('flags the classic mistakes', () => {
    const f = ids(`script-src 'self' 'unsafe-inline' 'unsafe-eval' https: ajax.googleapis.com http://cdn.example`)
    expect(f).toEqual(expect.arrayContaining(['unsafeInline', 'unsafeEval', 'scriptWildcard', 'bypassHost', 'insecureSource', 'objectSrc', 'baseUri', 'noReporting']))
    expect(ids(`default-src 'none'`)).not.toContain('objectSrc')
    expect(ids(`script-src 'strict-dynamic'`)).toContain('strictDynamicNoNonce')
    expect(ids(`object-src 'none'`)).toContain('noScriptSrc')
    expect(ids(`script-src 'self'; script-src *; plugin-types x; foo-src a`)).toEqual(expect.arrayContaining(['duplicate', 'deprecated', 'unknown']))
    expect(ids(`script-src 'nonce-abc'`)).toContain('weakNonce')
    expect(ids('')).toEqual(['empty'])
  })
  it('renders outputs', () => {
    expect(render(`default-src 'self'`, 'header', true)).toBe(`Content-Security-Policy-Report-Only: default-src 'self'`)
    expect(render(`default-src 'self'; frame-ancestors 'none'`, 'meta', false)).toBe(`<meta http-equiv="Content-Security-Policy" content="default-src 'self'">`)
    expect(render(`default-src 'self'`, 'nginx', false)).toBe(`add_header Content-Security-Policy "default-src 'self'" always;`)
    expect(newNonce()).toMatch(/^[\w+/]{22}==$/)
  })
})
