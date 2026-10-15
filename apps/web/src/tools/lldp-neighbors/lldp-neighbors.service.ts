import type { components } from '@/api/schema'

export type LldpNeighbor = components['schemas']['LldpNeighbor']

/** Linux interface names: up to 15 chars, no slashes or spaces. Empty = default route. */
export const isInterfaceName = (s: string) => /^$|^[\w.:-]{1,15}$/.test(s.trim())

/** "switch port" summary: system name, else chassis id. */
export const deviceName = (n: LldpNeighbor) => n.systemName || n.chassisId || n.sourceMac
