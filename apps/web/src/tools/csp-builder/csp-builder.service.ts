export interface Directive { name: string, values: string[] }
export type Severity = 'bad' | 'warn' | 'info'
export interface Finding { id: string, severity: Severity, directive: string, detail: string }

export const FETCH_DIRECTIVES = ['default-src', 'script-src', 'script-src-elem', 'script-src-attr', 'style-src', 'style-src-elem', 'style-src-attr', 'img-src', 'font-src', 'connect-src', 'media-src', 'object-src', 'frame-src', 'child-src', 'worker-src', 'manifest-src', 'fenced-frame-src']
export const OTHER_DIRECTIVES = ['base-uri', 'form-action', 'frame-ancestors', 'sandbox', 'upgrade-insecure-requests', 'report-to', 'report-uri', 'require-trusted-types-for', 'trusted-types', 'webrtc']
const DEPRECATED = ['block-all-mixed-content', 'plugin-types', 'prefetch-src', 'referrer', 'reflected-xss', 'navigate-to', 'disown-opener']
const KNOWN = new Set([...FETCH_DIRECTIVES, ...OTHER_DIRECTIVES, ...DEPRECATED])

/** Hosts that serve JSONP endpoints or old AngularJS, which defeat a host allowlist. */
export const BYPASS_HOSTS = ['*.googleapis.com', 'ajax.googleapis.com', 'cdnjs.cloudflare.com', 'cdn.jsdelivr.net', 'unpkg.com', '*.gstatic.com', 'www.google.com', 'raw.githubusercontent.com', '*.cloudfront.net', '*.amazonaws.com', 'accounts.google.com', 'www.googletagmanager.com']

export const PRESETS = {
  strict: `script-src 'nonce-{RANDOM}' 'strict-dynamic' https: 'unsafe-inline'; object-src 'none'; base-uri 'none'; frame-ancestors 'self'; report-to csp-endpoint`,
  allowlist: `default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'self'; upgrade-insecure-requests`,
  lockdown: `default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'`,
} as const

/** Parses a policy; later duplicates are kept so the evaluator can flag them (browsers ignore them). */
export function parseCsp(policy: string): Directive[] {
  return policy.split(';').map(p => p.trim().split(/\s+/).filter(Boolean)).filter(p => p.length).map(([name, ...values]) => ({ name: name!.toLowerCase(), values }))
}

export function serialize(ds: Directive[]): string {
  return ds.map(d => [d.name, ...d.values].join(' ')).join('; ')
}

const isNonceOrHash = (v: string) => /^'(?:nonce-|sha(?:256|384|512)-)/i.test(v)

export function evaluate(policy: string): Finding[] {
  const out: Finding[] = []
  const add = (id: string, severity: Severity, directive = '', detail = '') => out.push({ id, severity, directive, detail })
  const ds = parseCsp(policy)
  if (!ds.length)
    return [{ id: 'empty', severity: 'bad', directive: '', detail: '' }]
  const seen = new Map<string, Directive>()
  for (const d of ds) {
    if (seen.has(d.name))
      add('duplicate', 'warn', d.name)
    else
      seen.set(d.name, d)
    if (DEPRECATED.includes(d.name))
      add('deprecated', 'warn', d.name)
    else if (!KNOWN.has(d.name))
      add('unknown', 'warn', d.name)
  }
  const get = (n: string) => seen.get(n)?.values.map(v => v.toLowerCase())
  const script = get('script-src') ?? get('default-src')
  const scriptName = seen.has('script-src') ? 'script-src' : 'default-src'
  if (!script) {
    add('noScriptSrc', 'bad')
  }
  else if (!(script.length === 1 && script[0] === '\'none\'')) {
    const nonce = script.some(isNonceOrHash)
    const strictDynamic = script.includes('\'strict-dynamic\'')
    if (script.includes('\'unsafe-inline\''))
      add(nonce ? 'unsafeInlineFallback' : 'unsafeInline', nonce ? 'info' : 'bad', scriptName)
    if (script.includes('\'unsafe-eval\''))
      add('unsafeEval', 'warn', scriptName)
    if (script.includes('\'unsafe-hashes\''))
      add('unsafeHashes', 'warn', scriptName)
    if (strictDynamic && !nonce)
      add('strictDynamicNoNonce', 'bad', scriptName)
    if (!strictDynamic) {
      const wide = script.filter(v => v === '*' || v === 'https:' || v === 'http:' || v === 'data:' || v === 'blob:')
      if (wide.length)
        add('scriptWildcard', 'bad', scriptName, wide.join(' '))
      const bypass = script.filter(v => BYPASS_HOSTS.includes(v.replace(/^https?:\/\//, '').replace(/\/.*$/, '')))
      if (bypass.length)
        add('bypassHost', 'warn', scriptName, bypass.join(' '))
    }
    else if (script.some(v => !v.startsWith('\''))) {
      add('strictDynamicHosts', 'info', scriptName)
    }
    if (!nonce && !strictDynamic && script.includes('\'self\''))
      add('selfOnly', 'info', scriptName)
  }
  for (const [name, d] of seen) {
    const insecure = d.values.filter(v => v.toLowerCase().startsWith('http://'))
    if (insecure.length)
      add('insecureSource', 'warn', name, insecure.join(' '))
  }
  const objectSrc = get('object-src') ?? get('default-src')
  if (!objectSrc || !(objectSrc.length === 1 && objectSrc[0] === '\'none\''))
    add('objectSrc', 'bad', 'object-src')
  if (!seen.has('base-uri'))
    add('baseUri', 'warn', 'base-uri')
  if (!seen.has('frame-ancestors'))
    add('frameAncestors', 'info', 'frame-ancestors')
  if (!seen.has('form-action'))
    add('formAction', 'info', 'form-action')
  if (!seen.has('report-to') && !seen.has('report-uri'))
    add('noReporting', 'info')
  for (const v of [...(get('script-src') ?? []), ...(get('style-src') ?? [])]) {
    const m = /^'nonce-(.*)'$/.exec(v)
    if (m && m[1] !== '{random}' && m[1]!.length < 22)
      add('weakNonce', 'warn', '', v)
  }
  return out
}

/** 128 random bits, base64 (the length CSP Level 3 recommends at minimum). */
export function newNonce(): string {
  const b = crypto.getRandomValues(new Uint8Array(16))
  return btoa(String.fromCharCode(...b))
}

export type Output = 'header' | 'nginx' | 'apache' | 'meta'
export function render(policy: string, out: Output, reportOnly: boolean): string {
  const name = reportOnly ? 'Content-Security-Policy-Report-Only' : 'Content-Security-Policy'
  const p = serialize(parseCsp(policy))
  switch (out) {
    case 'header':
      return `${name}: ${p}`
    case 'nginx':
      return `add_header ${name} "${p.replaceAll('"', '\\"')}" always;`
    case 'apache':
      return `Header always set ${name} "${p.replaceAll('"', '\\"')}"`
    case 'meta': // frame-ancestors, report-* and sandbox are ignored in <meta>; Report-Only is not allowed
      return `<meta http-equiv="Content-Security-Policy" content="${serialize(parseCsp(policy).filter(d => !['frame-ancestors', 'report-to', 'report-uri', 'sandbox'].includes(d.name))).replaceAll('"', '&quot;')}">`
  }
}
