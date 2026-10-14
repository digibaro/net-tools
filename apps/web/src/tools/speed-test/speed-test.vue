<script setup lang="ts">
import { NAlert, NButton, NProgress } from 'naive-ui'
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiPost } from '@/api/client'
import ResultActions from '@/components/ResultActions.vue'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { probe, stats } from '../browser-latency/browser-latency.service'
import { DOWNLOAD_BYTES, drain, mbps, nextUploadSize, PHASE_MS, randomPayload, STREAMS, TOKEN_HEADER, UPLOAD_MAX_BYTES, UPLOAD_MIN_BYTES } from './speed-test.service'

type Phase = 'idle' | 'latency' | 'download' | 'upload' | 'done' | 'error'
const { t } = useI18n()
const phase = ref<Phase>('idle')
const latency = ref<ReturnType<typeof stats>>()
const download = ref(0)
const upload = ref(0)
const live = ref(0)
const failure = ref('')
let controller: AbortController | undefined

async function measureDownload(signal: AbortSignal, headers: Record<string, string>) {
  let bytes = 0
  const start = performance.now()
  const stop = AbortSignal.any([signal, AbortSignal.timeout(PHASE_MS)])
  await Promise.all(Array.from({ length: STREAMS }, async () => {
    try {
      while (!stop.aborted) { // keep each stream busy for the whole phase
        const res = await fetch(`/api/v1/speedtest/download?bytes=${DOWNLOAD_BYTES}&_=${Math.random()}`, { cache: 'no-store', signal: stop, headers })
        if (!res.ok)
          throw new Error(`HTTP ${res.status}`)
        await drain(res, (n) => {
          bytes += n
          live.value = mbps(bytes, performance.now() - start)
        })
      }
    }
    catch (e) {
      if (!stop.aborted)
        throw e
    }
  }))
  return mbps(bytes, performance.now() - start)
}

async function measureUpload(signal: AbortSignal, headers: Record<string, string>) {
  const payload = new Blob([randomPayload(UPLOAD_MAX_BYTES)])
  let bytes = 0
  const start = performance.now()
  const deadline = start + PHASE_MS
  await Promise.all(Array.from({ length: STREAMS }, async () => {
    let size = UPLOAD_MIN_BYTES
    while (performance.now() < deadline && !signal.aborted) {
      const t0 = performance.now()
      const res = await fetch('/api/v1/speedtest/upload', { method: 'POST', body: payload.slice(0, size), cache: 'no-store', signal, headers: { ...headers, 'Content-Type': 'application/octet-stream' } })
      if (!res.ok)
        throw new Error(`HTTP ${res.status}`)
      await res.arrayBuffer()
      bytes += size
      live.value = mbps(bytes, performance.now() - start)
      size = nextUploadSize(size, performance.now() - t0)
    }
  }))
  return mbps(bytes, performance.now() - start)
}

async function run() {
  controller?.abort()
  const c = new AbortController()
  controller = c
  failure.value = ''
  download.value = upload.value = live.value = 0
  latency.value = undefined
  try {
    phase.value = 'latency'
    // One session per test: the server rate-limits the session, not each of its many requests.
    const session = await apiPost<{ token: string }>('/v1/speedtest/session', {})
    const headers = { [TOKEN_HEADER]: session.token }
    await probe('/api/v1/speedtest/ping', c.signal, undefined, headers) // warm-up: opens the connection
    const samples: (number | null)[] = []
    for (let i = 0; i < 10; i++)
      samples.push(await probe('/api/v1/speedtest/ping', c.signal, undefined, headers))
    latency.value = stats(samples)
    phase.value = 'download'
    download.value = await measureDownload(c.signal, headers)
    live.value = 0
    phase.value = 'upload'
    upload.value = await measureUpload(c.signal, headers)
    phase.value = 'done'
  }
  catch (e) {
    if (c.signal.aborted)
      return
    failure.value = e instanceof Error ? e.message : String(e)
    phase.value = 'error'
  }
}
function stop() {
  controller?.abort()
  phase.value = 'idle'
}
onBeforeUnmount(() => controller?.abort())

const running = computed(() => ['latency', 'download', 'upload'].includes(phase.value))
const rows = computed(() => [
  { label: t('tools.speed-test.download'), value: download.value ? `${download.value} Mbit/s` : '—' },
  { label: t('tools.speed-test.upload'), value: upload.value ? `${upload.value} Mbit/s` : '—' },
  { label: t('tools.speed-test.latency'), value: latency.value?.received ? `${latency.value.median} ms (${t('ping.jitter')} ${latency.value.jitter} ms)` : '—' },
])
</script>

<template>
  <CCard>
    <p class="mt-0 text-sm">
      {{ t('tools.speed-test.intro') }}
    </p>
    <div class="flex gap-2">
      <NButton v-if="!running" type="primary" data-testid="run" @click="run">
        {{ phase === 'idle' ? t('tools.speed-test.start') : t('tools.speed-test.again') }}
      </NButton>
      <NButton v-else @click="stop">
        {{ t('common.stop') }}
      </NButton>
    </div>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.speed-test.note') }}
    </p>
  </CCard>
  <NAlert v-if="phase === 'error'" type="error" class="mb-4" data-testid="api-error">
    {{ t('tools.speed-test.failed', { error: failure }) }}
  </NAlert>
  <CCard v-if="phase !== 'idle'" :title="t('common.result')" data-testid="result">
    <div v-if="running" class="mb-4">
      <div class="mb-1 text-sm">
        {{ t(`tools.speed-test.phase.${phase}`) }}<template v-if="live">
          · {{ live }} Mbit/s
        </template>
      </div>
      <NProgress type="line" :percentage="phase === 'latency' ? 15 : phase === 'download' ? 50 : 85" :show-indicator="false" :aria-label="t(`tools.speed-test.phase.${phase}`)" />
    </div>
    <CKeyValueList :rows="rows" />
    <div v-if="phase === 'done'" class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="{ downloadMbps: download, uploadMbps: upload, latency }" filename="speed-test.json" />
    </div>
  </CCard>
</template>
