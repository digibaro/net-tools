/** An http(s) URL as the API accepts it (scheme optional, defaults to https). */
export function isValidHttpUrl(input: string): boolean {
  const s = input.trim()
  if (!s)
    return false
  try {
    const u = new URL(s.includes('://') ? s : `https://${s}`)
    return (u.protocol === 'http:' || u.protocol === 'https:') && !!u.hostname && !u.username && !u.password
  }
  catch {
    return false
  }
}
