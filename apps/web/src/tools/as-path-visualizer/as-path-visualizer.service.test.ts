import { describe, expect, it } from 'vitest'
import { buildGraph, classify, parsePath, parsePaths } from './as-path-visualizer.service'

describe('as-path-visualizer', () => {
  it('parses prepends, sets and origin codes', () => {
    const p = parsePath('AS3356 1299 64500 64500 64500 13335 i')!
    expect(p.length).toBe(6)
    expect(p.origin).toBe(13335)
    expect(p.prepends).toEqual([{ asn: 64500, count: 3, atOrigin: false }])
    const o = parsePath('174 13335 13335')!
    expect(o.prepends[0]!.atOrigin).toBe(true)
    const s = parsePath('174 {65001,65002, 65001}')!
    expect(s.hops[1]!.set).toEqual([65001, 65002])
    expect(s.length).toBe(2)
    expect(s.origin).toBeUndefined()
    expect(parsePath('nothing here')).toBeUndefined()
  })
  it('flags loops, private and reserved ASNs', () => {
    const p = parsePath('100 200 100 300 23456 65000')!
    expect(p.loops).toEqual([100])
    expect(p.privateAsns).toEqual([65000])
    expect(p.reservedAsns).toEqual([23456])
    expect(classify(4200000001)).toBe('private')
    expect(classify(64500)).toBe('documentation')
    expect(classify(13335)).toBe('public')
  })
  it('lays out shared upstreams', () => {
    const g = buildGraph(parsePaths('3356 1299 13335\n174 13335 13335\n6939 1299 13335'))
    const layerOf = (a: number) => g.nodes.find(n => n.asn === a)!.layer
    expect(layerOf(13335)).toBe(0)
    expect(layerOf(1299)).toBe(1)
    expect(layerOf(174)).toBe(1)
    expect(layerOf(3356)).toBe(2)
    expect(g.layers).toBe(3)
    expect(g.edges.find(e => e.from === 1299)!.paths).toBe(2)
    expect(g.nodes.find(n => n.asn === 13335)!.prepend).toBe(2)
    expect(g.nodes.find(n => n.asn === 13335)!.isOrigin).toBe(true)
  })
})
