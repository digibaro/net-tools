import type { Ref } from 'vue'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

type Primitive = string | number | boolean

export function parseQueryValue<T extends Primitive>(raw: unknown, defaultValue: T): T {
  const value = Array.isArray(raw) ? raw[0] : raw
  if (typeof value !== 'string')
    return defaultValue
  if (typeof defaultValue === 'number') {
    const n = Number(value)
    return (value.trim() !== '' && Number.isFinite(n) ? n : defaultValue) as T
  }
  if (typeof defaultValue === 'boolean')
    return (value === 'true' ? true : value === 'false' ? false : defaultValue) as T
  return value as T
}

/**
 * Two-way binds a value to a URL query parameter so tool inputs are shareable.
 * The parameter is removed from the URL while the value equals its default.
 */
export function useQueryParam<T extends Primitive>({ name, defaultValue }: { name: string, defaultValue: T }): Ref<T> {
  const route = useRoute()
  const router = useRouter()

  return computed<T>({
    get: () => parseQueryValue(route.query[name], defaultValue),
    set: (value) => {
      const query = { ...route.query }
      if (value === defaultValue || value === '')
        delete query[name]
      else
        query[name] = String(value)
      void router.replace({ query })
    },
  })
}
