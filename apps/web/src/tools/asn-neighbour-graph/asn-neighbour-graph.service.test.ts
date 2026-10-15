import type { NeighbourGraph } from './asn-neighbour-graph.service'
import { describe, expect, it } from 'vitest'
import { GRAPH_W, layout, shortName } from './asn-neighbour-graph.service'

const n = (asn: number) => ({ asn, name: `AS${asn}`, power: 1 })

describe('asn-neighbour-graph', () => {
  it('places upstreams left and downstreams right', () => {
    const g = { upstreams: [n(1), n(2)], downstreams: [n(3)], uncertain: [] } as unknown as NeighbourGraph
    const l = layout(g)
    expect(l.nodes.filter(p => p.side === 'up').every(p => p.x < GRAPH_W / 2)).toBe(true)
    expect(l.nodes.find(p => p.side === 'down')!.y).toBe(l.cy)
    expect(shortName('A'.repeat(40))).toHaveLength(26)
  })
})
