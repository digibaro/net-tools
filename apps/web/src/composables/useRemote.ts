import type { components } from '@/api/schema'
import { computed, ref, shallowRef } from 'vue'
import { useSse } from './useSse'

export type RemoteType = components['schemas']['RemoteMeta']['type']
export type RemoteMeta = components['schemas']['RemoteMeta']
export type RemoteProbeResult = components['schemas']['RemoteProbeResult']
export type RemoteSummary = components['schemas']['RemoteSummary']

/** Runs a Globalping measurement through /api/v1/remote/stream and collects probe results. */
export function useRemote() {
  const meta = shallowRef<RemoteMeta>()
  const probes = ref<RemoteProbeResult[]>([])
  const summary = shallowRef<RemoteSummary>()
  const sse = useSse('/v1/remote/stream', {
    meta: d => (meta.value = d as RemoteMeta),
    probe: d => probes.value.push(d as RemoteProbeResult),
    done: d => (summary.value = (d as { summary: RemoteSummary }).summary),
  })
  function start(params: { type: RemoteType, target: string, locations?: string, limit?: number, dnsType?: string, protocol?: string }) {
    meta.value = undefined
    probes.value = []
    summary.value = undefined
    sse.start(params)
  }
  const sorted = computed(() => [...probes.value].sort((a, b) => a.index - b.index))
  return { meta, probes: sorted, summary, status: sse.status, error: sse.error, start, stop: sse.stop }
}
