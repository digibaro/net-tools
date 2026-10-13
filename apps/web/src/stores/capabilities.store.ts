import type { Capabilities } from '@/api/client'
import type { Tool } from '@/tools/tool'
import { defineStore } from 'pinia'
import { ref, shallowRef } from 'vue'
import { getCapabilities } from '@/api/client'

export type CapabilityState = 'loading' | 'ready' | 'unreachable'

/** Why a tool cannot run, or undefined when it can. Browser tools never depend on the API. */
export function disabledReason(tool: Pick<Tool, 'runMode' | 'capability' | 'requiresApi' | 'requiresLan'>, state: CapabilityState, caps: Capabilities | undefined): string | undefined {
  const needsApi = tool.requiresApi || tool.runMode === 'server' || tool.runMode === 'public-api' || !!tool.capability
  if (!needsApi)
    return undefined
  if (state === 'unreachable')
    return 'api_unreachable'
  if (state === 'loading' || !caps)
    return undefined
  if (tool.requiresLan && !caps.mode.lan)
    return 'requires_lan_mode'
  if (!tool.capability)
    return undefined
  const cap = caps.tools[tool.capability]
  if (!cap)
    return 'unavailable'
  return cap.enabled ? undefined : (cap.reason ?? 'unavailable')
}

export const useCapabilitiesStore = defineStore('capabilities', () => {
  const state = ref<CapabilityState>('loading')
  const capabilities = shallowRef<Capabilities>()
  let pending: Promise<void> | undefined

  function load(): Promise<void> {
    pending ??= getCapabilities()
      .then((c) => {
        capabilities.value = c
        state.value = 'ready'
      })
      .catch(() => {
        state.value = 'unreachable'
        pending = undefined
      })
    return pending
  }

  const reasonFor = (tool: Tool) => disabledReason(tool, state.value, capabilities.value)

  return { state, capabilities, load, reasonFor }
})
