import { computed } from 'vue'
import { useCapabilitiesStore } from '@/stores/capabilities.store'
import { useQueryParam } from './useQueryParam'

/** "This server" vs "Worldwide" (Globalping) for tools that support both; state lives in the URL. */
export function useVantage() {
  const caps = useCapabilitiesStore()
  const vantage = useQueryParam({ name: 'vantage', defaultValue: 'server' })
  const locations = useQueryParam({ name: 'locations', defaultValue: '' })
  const limit = useQueryParam({ name: 'probes', defaultValue: 3 })
  const available = computed(() => caps.capabilities?.tools['remote.globalping']?.enabled ?? false)
  const worldwide = computed(() => vantage.value === 'worldwide' && available.value)
  return { vantage, locations, limit, available, worldwide }
}
