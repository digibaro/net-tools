<script setup lang="ts">
import type { LogEntry } from './websocket-tester.service'
import { NButton, NTag } from 'naive-ui'
import { computed, nextTick, onBeforeUnmount, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import CopyButton from '@/components/CopyButton.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { closeKey, hexPreview, isWsUrl, parseProtocols, pushLog } from './websocket-tester.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: 'wss://echo.websocket.org' })
const protocols = useQueryParam({ name: 'protocols', defaultValue: '' })
const validation = useValidation({ source: url, rules: [{ validator: isWsUrl, message: t('tools.websocket-tester.invalid') }] })
const message = ref('')
const log = ref<LogEntry[]>([])
const ws = shallowRef<WebSocket>()
const state = ref<'closed' | 'connecting' | 'open'>('closed')
const info = ref('')
let started = 0
const logEl = ref<HTMLElement>()

function add(dir: LogEntry['dir'], text: string, bytes = 0) {
  log.value = pushLog(log.value, { dir, at: Date.now(), text, bytes })
  void nextTick(() => logEl.value?.scrollTo({ top: logEl.value.scrollHeight }))
}

function connect() {
  if (!validation.value.isValid)
    return
  disconnect()
  try {
    const socket = new WebSocket(url.value.trim(), parseProtocols(protocols.value))
    socket.binaryType = 'arraybuffer'
    ws.value = socket
    state.value = 'connecting'
    started = performance.now()
    add('sys', t('tools.websocket-tester.events.connecting', { url: url.value.trim() }))
    socket.onopen = () => {
      state.value = 'open'
      info.value = [socket.protocol && `protocol ${socket.protocol}`, socket.extensions].filter(Boolean).join(' · ')
      add('sys', t('tools.websocket-tester.events.open', { ms: Math.round(performance.now() - started), info: info.value || '—' }))
    }
    socket.onmessage = (ev: MessageEvent) => {
      if (typeof ev.data === 'string')
        add('in', ev.data, new Blob([ev.data]).size)
      else
        add('in', `[binary] ${hexPreview(ev.data as ArrayBuffer)}`, (ev.data as ArrayBuffer).byteLength)
    }
    socket.onerror = () => add('sys', t('tools.websocket-tester.events.error'))
    socket.onclose = (ev: CloseEvent) => {
      if (ws.value === socket) {
        state.value = 'closed'
        ws.value = undefined
      }
      add('sys', t('tools.websocket-tester.events.close', { code: ev.code, meaning: t(`tools.websocket-tester.close.${closeKey(ev.code)}`), reason: ev.reason || '—' }))
    }
  }
  catch (e) {
    add('sys', e instanceof Error ? e.message : String(e))
    state.value = 'closed'
  }
}

function send() {
  if (ws.value?.readyState !== WebSocket.OPEN || !message.value)
    return
  ws.value.send(message.value)
  add('out', message.value, new Blob([message.value]).size)
}

function disconnect() {
  ws.value?.close(1000, 'closed by user')
}
onBeforeUnmount(() => ws.value?.close(1001))

const stateType = { closed: 'default', connecting: 'warning', open: 'success' } as const
const dirLabel = { in: '←', out: '→', sys: '•' } as const
const time = (at: number) => new Date(at).toLocaleTimeString()
const asText = computed(() => log.value.map(e => `${time(e.at)} ${dirLabel[e.dir]} ${e.text}`).join('\n'))
</script>

<template>
  <CCard>
    <div class="grid gap-x-4 sm:grid-cols-[1fr_14rem]">
      <CInputText v-model:value="url" :label="t('tools.websocket-tester.url')" placeholder="wss://example.com/socket" :validation="validation" monospace test-id="url" />
      <CInputText v-model:value="protocols" :label="t('tools.websocket-tester.protocols')" placeholder="graphql-ws" monospace />
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <NButton v-if="state === 'closed'" type="primary" :disabled="!validation.isValid" data-testid="connect" @click="connect">
        {{ t('tools.websocket-tester.connect') }}
      </NButton>
      <NButton v-else data-testid="disconnect" @click="disconnect">
        {{ t('tools.websocket-tester.disconnect') }}
      </NButton>
      <NTag :type="stateType[state]" :bordered="false" data-testid="state">
        {{ t(`tools.websocket-tester.state.${state}`) }}
      </NTag>
      <span v-if="state === 'open' && info" class="text-xs nt-muted">{{ info }}</span>
    </div>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.websocket-tester.note') }}
    </p>
  </CCard>
  <CCard :title="t('tools.websocket-tester.messages')" data-testid="result">
    <form class="contents" @submit.prevent="send">
      <CInputText v-model:value="message" :label="t('tools.websocket-tester.message')" multiline :rows="2" monospace test-id="message" />
      <NButton attr-type="submit" :disabled="state !== 'open' || !message" data-testid="send">
        {{ t('tools.websocket-tester.send') }}
      </NButton>
    </form>
    <div ref="logEl" class="mt-3 max-h-96 overflow-auto text-xs font-mono" tabindex="0" data-testid="log">
      <div v-for="(e, i) in log" :key="i" class="flex gap-2 py-0.5" :class="e.dir === 'sys' ? 'nt-muted' : ''">
        <span class="shrink-0">{{ time(e.at) }}</span>
        <span class="shrink-0" :aria-label="t(`tools.websocket-tester.dir.${e.dir}`)">{{ dirLabel[e.dir] }}</span>
        <span class="whitespace-pre-wrap break-all">{{ e.text }}</span>
      </div>
    </div>
    <div class="mt-3 flex gap-2">
      <CopyButton :text="() => asText" :label="t('common.copyText')" />
      <NButton size="small" secondary :disabled="!log.length" @click="log = []">
        {{ t('tools.websocket-tester.clear') }}
      </NButton>
    </div>
  </CCard>
</template>
