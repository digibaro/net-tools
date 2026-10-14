import type { components } from '@/api/schema'
import { humanize } from '@/utils/units'

export type Compression = components['schemas']['Compression']

export const size = (bytes: number) => humanize(bytes * 8, 'size-bytes')

/** "−82 %" for a ratio of 0.18. */
export const saving = (ratio: number) => (ratio > 0 && ratio < 1 ? `−${Math.round((1 - ratio) * 100)} %` : '')
