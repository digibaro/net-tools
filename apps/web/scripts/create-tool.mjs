#!/usr/bin/env node
// Scaffolds a tool (docs/TOOL_AUTHORING.md):
//   node scripts/create-tool.mjs <kebab-name> [--server] [--category <id>]
// Creates src/tools/<name>/ (index, component, service, unit + e2e test), registers it in
// src/tools/index.ts and adds i18n keys to locales/en.yml. --server also scaffolds
// apps/api/internal/tools/<name>/.
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const args = process.argv.slice(2)
const name = args.find(a => !a.startsWith('--') && args[args.indexOf(a) - 1] !== '--category')
const server = args.includes('--server')
const category = args.includes('--category') ? args[args.indexOf('--category') + 1] : 'ip'

if (!name || !/^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/.test(name)) {
  console.error('usage: create-tool.mjs <kebab-name> [--server] [--category <id>]')
  process.exit(1)
}

const camel = name.replace(/-([a-z0-9])/g, (_, c) => c.toUpperCase())
const title = name.split('-').map(w => w[0].toUpperCase() + w.slice(1)).join(' ')
const today = new Date().toISOString().slice(0, 10)
const dir = join(root, 'src/tools', name)
if (existsSync(dir)) {
  console.error(`${dir} already exists`)
  process.exit(1)
}

const indexPath = join(root, 'src/tools/index.ts')
let index = readFileSync(indexPath, 'utf8')
const catRe = new RegExp(`(\\{ id: '${category}', components: \\[)([^\\]]*)\\]`)
if (!catRe.test(index)) {
  console.error(`unknown category '${category}'`)
  process.exit(1)
}

const files = {
  'index.ts': `import { Tool as ToolIcon } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.${name}.title'),
  path: '/${name}',
  description: translate('tools.${name}.description'),
  keywords: ['${name.split('-').join('\', \'')}'],
  component: () => import('./${name}.vue'),
  icon: ToolIcon,
  createdAt: new Date('${today}'),
  runMode: '${server ? 'server' : 'browser'}',
})
`,
  [`${name}.service.ts`]: `export function example(input: string): string {
  return input.trim()
}
`,
  [`${name}.service.test.ts`]: `import { describe, expect, it } from 'vitest'
import { example } from './${name}.service'

describe('${name}', () => {
  it('works', () => {
    expect(example(' x ')).toBe('x')
  })
})
`,
  [`${name}.vue`]: `<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { example } from './${name}.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'q', defaultValue: '' })
const result = computed(() => example(input.value))
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.${name}.input')" test-id="input" />
  </CCard>
  <CCard v-if="result" :title="t('common.result')">
    <span data-testid="result">{{ result }}</span>
  </CCard>
</template>
`,
  [`${name}.e2e.spec.ts`]: `import { expect, test } from '@playwright/test'

test('${name}: input from the URL produces a result', async ({ page }) => {
  await page.goto('/${name}?q=x')
  await expect(page.getByTestId('result')).toHaveText('x')
})
`,
}

mkdirSync(dir, { recursive: true })
for (const [file, content] of Object.entries(files))
  writeFileSync(join(dir, file), content)

const importLine = `import { tool as ${camel} } from './${name}'\n`
index = index.replace(/^(import [^\n]+\n)(?!import )/m, `$1${importLine}`)
index = index.replace(catRe, (_, head, list) => `${head}${list.trim() ? `${list.trim()}, ` : ''}${camel}]`)
writeFileSync(indexPath, index)

const localePath = join(root, 'locales/en.yml')
writeFileSync(localePath, `${readFileSync(localePath, 'utf8').trimEnd()}
  ${name}:
    title: ${title}
    description: TODO describe ${title}
    input: Input
`)

if (server) {
  const pkg = name.replace(/-/g, '')
  const goDir = join(root, '../api/internal/tools', pkg)
  mkdirSync(goDir, { recursive: true })
  writeFileSync(join(goDir, 'register.go'), `// Package ${pkg} implements the ${title} tool.
package ${pkg}

import (
	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/config"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
)

// Tool implements tools.Tool.
type Tool struct{}

// ID is the capability id.
func (Tool) ID() string { return "TODO.${pkg}" }

// Enabled reports availability.
func (Tool) Enabled(config.Config) (bool, tools.Reason) { return true, "" }

// Mount registers routes under /api/v1.
func (t Tool) Mount(r chi.Router, _ tools.Deps) {
	r.Get("/${name}", t.handle)
}
`)
  writeFileSync(join(goDir, 'handler.go'), `package ${pkg}

import (
	"net/http"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
)

func (Tool) handle(w http.ResponseWriter, r *http.Request) {
	httpapi.Error(w, r, httpapi.NewError(httpapi.CodeInternal, "not implemented"))
}
`)
  console.log(`scaffolded apps/api/internal/tools/${pkg} — add it to tools.All() and openapi.yaml`)
}

console.log(`scaffolded src/tools/${name} in category '${category}'`)
