export function downloadText(filename: string, text: string, type = 'application/json') {
  const url = URL.createObjectURL(new Blob([text], { type }))
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.append(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 0)
}

export function toJson(value: unknown): string {
  return `${JSON.stringify(value, (_k, v: unknown) => (typeof v === 'bigint' ? v.toString() : v), 2)}\n`
}
