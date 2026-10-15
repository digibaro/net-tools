import type { Component } from 'vue'
import { translate } from '@/plugins/i18n.plugin'

export type RunMode = 'browser' | 'server' | 'public-api' | 'hybrid'

export interface ToolDefinition {
  /** Translated display name. */
  name: string
  /** Route path, e.g. '/dns-lookup'. */
  path: string
  description: string
  keywords: string[]
  component: () => Promise<Component>
  icon: Component
  /** Drives the "new" badge. */
  createdAt?: Date
  runMode: RunMode
  /** Capability id that must be enabled in /api/v1/capabilities, e.g. 'dns.lookup'. */
  capability?: string
  /** Needs the API at all (server, hybrid without browser fallback, system endpoints). */
  requiresApi?: boolean
  /** Public providers offered as an option, e.g. ['globalping']. */
  providers?: string[]
  requiresLan?: boolean
}

export interface Tool extends ToolDefinition {
  isNew: boolean
}

export interface ToolCategory {
  /** i18n key suffix under `categories.` */
  id: string
  name: string
  components: Tool[]
}

/**
 * The latest release, set at build time from the release tag (vite.config.ts). Tools whose
 * `createdAt` is later carry the "new" badge until the next release.
 */
export const LATEST_RELEASE = { version: __LATEST_RELEASE__.version, date: new Date(__LATEST_RELEASE__.date) }

export function isNew(createdAt: Date | undefined, release = LATEST_RELEASE.date): boolean {
  return !!createdAt && createdAt.getTime() > release.getTime()
}

/**
 * Name and description are looked up on every access (keys `tools.<path>.title|description`), so
 * they follow the active language; the strings passed in only serve as the build-time default.
 */
export function defineTool(tool: ToolDefinition): Tool {
  const slug = tool.path.slice(1)
  const out = { ...tool, isNew: isNew(tool.createdAt) }
  Object.defineProperties(out, {
    name: { get: () => translate(`tools.${slug}.title`), enumerable: true },
    description: { get: () => translate(`tools.${slug}.description`), enumerable: true },
  })
  return out
}
