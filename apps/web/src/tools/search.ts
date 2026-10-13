import type { Tool } from './tool'
import Fuse from 'fuse.js'

export function createToolSearch(tools: Tool[]) {
  const fuse = new Fuse(tools, {
    keys: [
      { name: 'name', weight: 3 },
      { name: 'keywords', weight: 2 },
      { name: 'description', weight: 1 },
    ],
    threshold: 0.35,
    ignoreLocation: true,
  })
  return (query: string, limit = 20): Tool[] => {
    const q = query.trim()
    return q ? fuse.search(q, { limit }).map(r => r.item) : []
  }
}
