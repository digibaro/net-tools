import { useClipboard } from '@vueuse/core'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'

export function useCopy() {
  const { copy: copyToClipboard, isSupported } = useClipboard({ legacy: true })
  const message = useMessage()
  const { t } = useI18n()

  async function copy(text: string) {
    try {
      await copyToClipboard(text)
      message.success(t('common.copied'))
    }
    catch {
      message.error(t('common.copyFailed'))
    }
  }

  return { copy, isSupported }
}
