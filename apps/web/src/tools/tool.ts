import type { Component } from 'vue'

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

const NEW_TOOL_DAYS = 14

export function isNew(createdAt: Date | undefined, now = new Date()): boolean {
  if (!createdAt)
    return false
  const ageMs = now.getTime() - createdAt.getTime()
  return ageMs >= 0 && ageMs < NEW_TOOL_DAYS * 24 * 60 * 60 * 1000
}

export function defineTool(tool: ToolDefinition): Tool {
  return { ...tool, isNew: isNew(tool.createdAt) }
}
