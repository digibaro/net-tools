import type { MaybeRefOrGetter } from 'vue'
import { computed, toValue } from 'vue'

export interface ValidationRule<T> {
  validator: (value: T) => boolean
  message: string
}

export interface Validation {
  isValid: boolean
  message: string
  status: 'error' | undefined
}

/**
 * Validates a value against rules. Empty strings are considered valid (nothing to show yet)
 * unless `required` is set; use `isValid && value !== ''` before running a tool.
 */
export function useValidation<T>({ source, rules, required = false }: {
  source: MaybeRefOrGetter<T>
  rules: MaybeRefOrGetter<ValidationRule<T>[]>
  required?: boolean
}) {
  return computed<Validation>(() => {
    const value = toValue(source)
    if (!required && (value === '' || value === undefined || value === null))
      return { isValid: true, message: '', status: undefined }
    for (const rule of toValue(rules)) {
      let ok = false
      try {
        ok = rule.validator(value)
      }
      catch {
        ok = false
      }
      if (!ok)
        return { isValid: false, message: rule.message, status: 'error' }
    }
    return { isValid: true, message: '', status: undefined }
  })
}
