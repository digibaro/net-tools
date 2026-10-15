import type { components } from '@/api/schema'

export type NeighbourGraph = components['schemas']['NeighbourGraph']
export type GraphNode = components['schemas']['GraphNode']

export const GRAPH_W = 900
export const ROW_H = 30
export const MAX_SHOWN = 15

export interface Placed { node: GraphNode, x: number, y: number, side: 'up' | 'down' }

/** Upstreams in the left column, downstreams in the right; the centre node is vertically centred. */
export function layout(g: NeighbourGraph, shown = MAX_SHOWN): { nodes: Placed[], height: number, cx: number, cy: number } {
  const ups = g.upstreams.slice(0, shown)
  const downs = g.downstreams.slice(0, shown)
  const rows = Math.max(ups.length, downs.length, 1)
  const height = rows * ROW_H + 20
  const place = (list: GraphNode[], x: number, side: Placed['side']) => {
    const top = (height - list.length * ROW_H) / 2 + ROW_H / 2
    return list.map((node, i) => ({ node, x, y: top + i * ROW_H, side }))
  }
  return { nodes: [...place(ups, 140, 'up'), ...place(downs, GRAPH_W - 140, 'down')], height, cx: GRAPH_W / 2, cy: height / 2 }
}

export const shortName = (n: string) => (n.length > 26 ? `${n.slice(0, 25)}…` : n)
