import type { Plugin } from 'vite'
import { createHash } from 'node:crypto'

const SCRIPT = /<script(\s[^>]*)?>([\s\S]*?)<\/script>/g

/** CSP source expressions ('sha256-…') for every inline <script> in an HTML document. */
export function inlineScriptHashes(html: string): string[] {
  return [...html.matchAll(SCRIPT)]
    .filter(([, attrs]) => !/\ssrc=/.test(attrs ?? ''))
    .map(([, , body]) => `'sha256-${createHash('sha256').update(body ?? '', 'utf8').digest('base64')}'`)
}

/**
 * Emits `csp-script-hashes.txt` next to index.html; docker/web.Dockerfile splices it into the
 * nginx Content-Security-Policy so only our own inline scripts may run.
 */
export function cspScriptHashes(): Plugin {
  return {
    name: 'nettools:csp-script-hashes',
    apply: 'build',
    enforce: 'post',
    generateBundle(_, bundle) {
      const index = bundle['index.html']
      if (!index || index.type !== 'asset')
        return
      const html = typeof index.source === 'string' ? index.source : new TextDecoder().decode(index.source)
      this.emitFile({ type: 'asset', fileName: 'csp-script-hashes.txt', source: `${inlineScriptHashes(html).join(' ')}\n` })
    },
  }
}
