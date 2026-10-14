/** Shell-like splitting of a curl command: quotes, backslash escapes, line continuations, $'…'. */
export function tokenize(input: string): string[] | undefined {
  const s = input.replace(/\\\r?\n/g, ' ')
  const out: string[] = []
  let cur = ''
  let has = false
  for (let i = 0; i < s.length; i++) {
    const c = s[i]!
    if (/\s/.test(c)) {
      if (has)
        out.push(cur)
      cur = ''
      has = false
      continue
    }
    has = true
    if (c === '\'') {
      const end = s.indexOf('\'', i + 1)
      if (end < 0)
        return undefined
      cur += s.slice(i + 1, end)
      i = end
    }
    else if (c === '$' && s[i + 1] === '\'') {
      let j = i + 2
      for (; j < s.length && s[j] !== '\''; j++) {
        if (s[j] === '\\' && j + 1 < s.length) {
          const e = s[++j]!
          cur += ({ 'n': '\n', 't': '\t', 'r': '\r', '\\': '\\', '\'': '\'' } as Record<string, string>)[e] ?? e
        }
        else {
          cur += s[j]
        }
      }
      if (j >= s.length)
        return undefined
      i = j
    }
    else if (c === '"') {
      let j = i + 1
      for (; j < s.length && s[j] !== '"'; j++) {
        if (s[j] === '\\' && '"\\$`'.includes(s[j + 1] ?? ''))
          j++
        cur += s[j]
      }
      if (j >= s.length)
        return undefined
      i = j
    }
    else if (c === '\\' && i + 1 < s.length) {
      cur += s[++i]
    }
    else {
      cur += c
    }
  }
  if (has)
    out.push(cur)
  return out
}

export interface CurlSpec {
  url: string
  method: string
  headers: [string, string][]
  data: string
  form: [string, string][]
  user: string
  insecure: boolean
  compressed: boolean
  follow: boolean
  verbose: boolean
  maxTime: number | null
  unknown: string[]
}

export const emptySpec = (): CurlSpec => ({ url: '', method: 'GET', headers: [], data: '', form: [], user: '', insecure: false, compressed: false, follow: false, verbose: false, maxTime: null, unknown: [] })

const WITH_ARG = new Set(['-X', '--request', '-H', '--header', '-d', '--data', '--data-raw', '--data-binary', '--data-ascii', '--data-urlencode', '--json', '-F', '--form', '-u', '--user', '-A', '--user-agent', '-e', '--referer', '-b', '--cookie', '-m', '--max-time', '-o', '--output', '--url', '-x', '--proxy', '--connect-timeout', '-c', '--cookie-jar', '-w', '--write-out', '-r', '--range'])

/** Parses `curl …` into a spec; unknown flags are kept so nothing is silently lost. */
export function parseCurl(input: string): CurlSpec | 'syntax' | 'not-curl' {
  const tokens = tokenize(input.trim())
  if (!tokens)
    return 'syntax'
  if (tokens[0] !== 'curl')
    return 'not-curl'
  const spec = emptySpec()
  let explicitMethod = ''
  let head = false
  let get = false
  const data: string[] = []
  const args = tokens.slice(1)
  for (let i = 0; i < args.length; i++) {
    let flag = args[i]!
    let val: string | undefined
    // --header=value and -XPOST forms
    if (flag.startsWith('--') && flag.includes('=')) {
      [flag, val] = [flag.slice(0, flag.indexOf('=')), flag.slice(flag.indexOf('=') + 1)]
    }
    else if (/^-[A-Z].+/i.test(flag) && WITH_ARG.has(flag.slice(0, 2))) {
      [flag, val] = [flag.slice(0, 2), flag.slice(2)]
    }
    if (WITH_ARG.has(flag) && val === undefined)
      val = args[++i] ?? ''
    switch (flag) {
      case '-X':
      case '--request':
        explicitMethod = val!.toUpperCase()
        break
      case '-H': case '--header': {
        const idx = val!.indexOf(':')
        if (idx > 0)
          spec.headers.push([val!.slice(0, idx).trim(), val!.slice(idx + 1).trim()])
        break
      }
      case '-d':
      case '--data':
      case '--data-raw':
      case '--data-binary':
      case '--data-ascii':
      case '--data-urlencode':
        data.push(val!)
        break
      case '--json':
        data.push(val!)
        spec.headers.push(['Content-Type', 'application/json'], ['Accept', 'application/json'])
        break
      case '-F': case '--form': {
        const idx = val!.indexOf('=')
        spec.form.push(idx > 0 ? [val!.slice(0, idx), val!.slice(idx + 1)] : [val!, ''])
        break
      }
      case '-u':
      case '--user':
        spec.user = val!
        break
      case '-A':
      case '--user-agent':
        spec.headers.push(['User-Agent', val!])
        break
      case '-e':
      case '--referer':
        spec.headers.push(['Referer', val!])
        break
      case '-b':
      case '--cookie':
        spec.headers.push(['Cookie', val!])
        break
      case '-m':
      case '--max-time':
        spec.maxTime = Number(val) || null
        break
      case '--url':
        spec.url = val!
        break
      case '-k':
      case '--insecure':
        spec.insecure = true
        break
      case '--compressed':
        spec.compressed = true
        break
      case '-L':
      case '--location':
        spec.follow = true
        break
      case '-v':
      case '--verbose':
        spec.verbose = true
        break
      case '-I':
      case '--head':
        head = true
        break
      case '-G':
      case '--get':
        get = true
        break
      case '-s': case '--silent': case '-S': case '--show-error': case '-f': case '--fail': case '-i': case '--include': break
      default:
        if (!flag.startsWith('-') && !spec.url)
          spec.url = flag
        else
          spec.unknown.push(val === undefined ? flag : `${flag} ${val}`)
    }
  }
  if (get && data.length) {
    spec.url += (spec.url.includes('?') ? '&' : '?') + data.join('&')
  }
  else {
    spec.data = data.join('&')
  }
  spec.method = explicitMethod || (head ? 'HEAD' : (spec.data || spec.form.length) && !get ? 'POST' : 'GET')
  return spec
}

/** POSIX single-quote a word when needed. */
export function quote(s: string): string {
  return /^[\w@%+=:,./-]+$/.test(s) ? s : `'${s.replace(/'/g, `'\\''`)}'`
}

/** Builds a curl command, one option per line. */
export function buildCurl(s: CurlSpec): string {
  const parts: string[] = ['curl']
  const defaultMethod = s.data || s.form.length ? 'POST' : 'GET'
  if (s.method && s.method !== defaultMethod)
    parts.push(s.method === 'HEAD' ? '-I' : `-X ${s.method}`)
  for (const [k, v] of s.headers)
    parts.push(`-H ${quote(`${k}: ${v}`)}`)
  if (s.data)
    parts.push(`--data-raw ${quote(s.data)}`)
  for (const [k, v] of s.form)
    parts.push(`-F ${quote(`${k}=${v}`)}`)
  if (s.user)
    parts.push(`-u ${quote(s.user)}`)
  if (s.follow)
    parts.push('-L')
  if (s.compressed)
    parts.push('--compressed')
  if (s.insecure)
    parts.push('-k')
  if (s.verbose)
    parts.push('-v')
  if (s.maxTime)
    parts.push(`--max-time ${s.maxTime}`)
  parts.push(quote(s.url || 'https://example.com'))
  return parts.join(' \\\n  ')
}
