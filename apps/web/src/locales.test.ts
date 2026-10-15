import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { parse } from 'yaml'

// Every locale must have exactly the keys of en.yml, with the same {placeholders} per message.
const dir = join(import.meta.dirname, '../locales')
const load = (file: string) => parse(readFileSync(join(dir, file), 'utf8')) as Record<string, unknown>

function flatten(obj: unknown, prefix = '', out = new Map<string, string>()): Map<string, string> {
  if (obj !== null && typeof obj === 'object') {
    for (const [k, v] of Object.entries(obj))
      flatten(v, prefix ? `${prefix}.${k}` : k, out)
  }
  else {
    out.set(prefix, String(obj))
  }
  return out
}

const placeholders = (s: string) => (s.match(/\{[^}]*\}/g) ?? []).sort()
const en = flatten(load('en.yml'))
const others = readdirSync(dir).filter(f => f.endsWith('.yml') && f !== 'en.yml')

describe('locales', () => {
  it('has at least one translation', () => {
    expect(others).toContain('nl.yml')
  })

  it.each(others)('%s has the same keys as en.yml', (file) => {
    const keys = [...flatten(load(file)).keys()]
    expect(keys.filter(k => !en.has(k)), 'extra keys').toEqual([])
    expect([...en.keys()].filter(k => !keys.includes(k)), 'missing keys').toEqual([])
  })

  it.each(others)('%s uses the same placeholders as en.yml', (file) => {
    const msgs = flatten(load(file))
    const wrong = [...en].filter(([k, v]) => msgs.has(k) && placeholders(v).join() !== placeholders(msgs.get(k)!).join())
    expect(wrong.map(([k]) => k)).toEqual([])
  })
})
