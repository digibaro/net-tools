<script setup lang="ts">
import type { Candidate } from './webrtc-leak-test.service'
import { NAlert, NButton, NTag } from 'naive-ui'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import CCard from '@/ui/c-card.vue'
import { fetchPublicIp } from '../what-is-my-ip/what-is-my-ip.service'
import { parseCandidate, STUN_SERVERS, verdict } from './webrtc-leak-test.service'

const { t } = useI18n()
const supported = typeof RTCPeerConnection !== 'undefined'
const running = ref(false)
const done = ref(false)
const candidates = ref<Candidate[]>([])
const httpIps = ref<string[]>([])
let pc: RTCPeerConnection | undefined

async function run() {
  if (!supported)
    return
  pc?.close()
  candidates.value = []
  done.value = false
  running.value = true
  const ipsPromise = Promise.all([fetchPublicIp(4), fetchPublicIp(6)])
  const conn = new RTCPeerConnection({ iceServers: [{ urls: STUN_SERVERS }] })
  pc = conn
  conn.createDataChannel('nettools')
  const finished = new Promise<void>((resolve) => {
    conn.onicecandidate = (ev) => {
      if (!ev.candidate) {
        resolve()
        return
      }
      const c = parseCandidate(ev.candidate.candidate)
      if (c && !candidates.value.some(x => x.address === c.address && x.type === c.type && x.protocol === c.protocol))
        candidates.value.push(c)
    }
    setTimeout(resolve, 8000) // gathering can stall behind firewalls
  })
  await conn.setLocalDescription(await conn.createOffer())
  await finished
  httpIps.value = (await ipsPromise).filter((x): x is string => !!x)
  conn.close()
  running.value = false
  done.value = true
}
onMounted(run)
onBeforeUnmount(() => pc?.close())

const v = computed(() => verdict(candidates.value, httpIps.value))
const rows = computed(() => candidates.value.map((c, i) => ({ id: i, type: t(`tools.webrtc-leak-test.types.${c.type}`, c.type), address: c.address, kind: t(`tools.webrtc-leak-test.kinds.${c.kind}`), protocol: c.protocol.toUpperCase() })))
const col = (k: string) => t(`tools.webrtc-leak-test.columns.${k}`)
</script>

<template>
  <NAlert v-if="!supported" type="info" class="mb-4" data-testid="result">
    {{ t('tools.webrtc-leak-test.unsupported') }}
  </NAlert>
  <template v-else>
    <CCard>
      <p class="mt-0 text-sm">
        {{ t('tools.webrtc-leak-test.intro') }}
      </p>
      <NButton :loading="running" data-testid="run" @click="run">
        {{ t('tools.webrtc-leak-test.run') }}
      </NButton>
    </CCard>
    <template v-if="done">
      <div class="mb-4 flex flex-col gap-2" data-testid="result">
        <NAlert :type="v.localLeak ? 'warning' : 'success'">
          {{ v.localLeak ? t('tools.webrtc-leak-test.localLeak') : t('tools.webrtc-leak-test.noLocalLeak') }}
        </NAlert>
        <NAlert v-if="v.mismatched.length" type="error" data-testid="vpn-leak">
          {{ t('tools.webrtc-leak-test.mismatch', { webrtc: v.mismatched.join(', '), http: httpIps.join(', ') }) }}
        </NAlert>
        <NAlert v-else-if="v.publicAddresses.length" type="info">
          {{ t('tools.webrtc-leak-test.publicSame', { list: v.publicAddresses.join(', ') }) }}
        </NAlert>
        <NAlert v-else type="info">
          {{ t('tools.webrtc-leak-test.noPublic') }}
        </NAlert>
      </div>
      <CCard :title="t('tools.webrtc-leak-test.candidates', { n: candidates.length })">
        <ResultTable v-if="rows.length" :columns="[{ key: 'type', title: col('type'), width: 160 }, { key: 'address', title: col('address'), mono: true, width: 300 }, { key: 'kind', title: col('kind'), width: 140 }, { key: 'protocol', title: col('protocol'), width: 90 }]" :rows="rows" row-key="id" :actions="false" />
        <div class="mt-3 flex flex-wrap items-center gap-2">
          <NTag v-for="ip in httpIps" :key="ip" size="small" :bordered="false">
            HTTP: {{ ip }}
          </NTag>
          <ResultActions :text="() => candidates.map(c => `${c.type}\t${c.address}\t${c.kind}`).join('\n')" :json="{ candidates, httpIps, verdict: v }" filename="webrtc.json" />
        </div>
      </CCard>
    </template>
  </template>
</template>
