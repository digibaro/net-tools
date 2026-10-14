#!/usr/bin/env node
// Regenerates bundled reference data from official registries.
//   node scripts/gen-data.mjs <dir with IANA CSVs>
// Expected files (from https://www.iana.org/assignments/…):
//   service-names-port-numbers.csv, protocol-numbers-1.csv, icmp-parameters-types.csv,
//   icmp-codes-<type>.csv (icmp-parameters-codes-<type>.csv), icmpv6-parameters-2.csv,
//   dns-parameters-4.csv
// The OUI table comes from the oui-data package (IEEE MA-L registry).
import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const src = process.argv[2]
if (!src) {
  console.error('usage: gen-data.mjs <dir with IANA CSVs>')
  process.exit(1)
}

/** RFC 4180 CSV parser (quoted fields, escaped quotes, embedded newlines). */
function parseCsv(text) {
  const rows = []
  let row = []
  let field = ''
  let quoted = false
  for (let i = 0; i < text.length; i++) {
    const ch = text[i]
    if (quoted) {
      if (ch === '"' && text[i + 1] === '"') {
        field += '"'
        i++
      }
      else if (ch === '"') {
        quoted = false
      }
      else {
        field += ch
      }
    }
    else if (ch === '"') {
      quoted = true
    }
    else if (ch === ',') {
      row.push(field)
      field = ''
    }
    else if (ch === '\n' || ch === '\r') {
      if (ch === '\r' && text[i + 1] === '\n')
        i++
      row.push(field)
      rows.push(row)
      row = []
      field = ''
    }
    else {
      field += ch
    }
  }
  if (field || row.length) {
    row.push(field)
    rows.push(row)
  }
  const [header, ...data] = rows
  return data.filter(r => r.some(Boolean)).map(r => Object.fromEntries(header.map((h, i) => [h, (r[i] ?? '').trim()])))
}

const csv = name => parseCsv(readFileSync(join(src, name), 'utf8'))
const clean = s => s.replace(/\s+/g, ' ').trim()
/** "[RFC792][RFC 1122]" → ["RFC 792", "RFC 1122"]; other references kept as text. */
function refs(s) {
  return [...s.matchAll(/\[([^\]]+)\]/g)]
    .map(m => /^RFC\s*(\d+)/i.exec(m[1]))
    .filter(Boolean)
    .map(m => `RFC ${m[1]}`)
}
function write(rel, data) {
  writeFileSync(join(root, rel), typeof data === 'string' ? data : `${JSON.stringify(data)}\n`)
  console.log(`wrote ${rel}`)
}
const out = dir => existsSync(join(root, dir))

// ── Ports ──────────────────────────────────────────────────────────────
if (out('src/tools/port-reference')) {
  const seen = new Set()
  const ports = []
  for (const r of csv('service-names-port-numbers.csv')) {
    if (!r['Port Number'] || !r['Transport Protocol'] || (/^(?:Unassigned|Reserved)$/i.test(r.Description) && !r['Service Name']))
      continue
    const key = `${r['Port Number']}/${r['Transport Protocol']}/${r['Service Name']}`
    if (seen.has(key))
      continue
    seen.add(key)
    ports.push([r['Port Number'], r['Transport Protocol'], r['Service Name'], clean(r.Description), refs(r.Reference)[0] ?? ''])
  }
  write('src/tools/port-reference/ports.json', ports)
}

// ── IP protocols and ICMP ──────────────────────────────────────────────
if (out('src/tools/protocol-reference')) {
  const protocols = csv('protocol-numbers-1.csv')
    .filter(r => /^\d+$/.test(r.Decimal))
    .map(r => ({ number: Number(r.Decimal), keyword: r.Keyword, name: clean(r.Protocol), ipv6ExtensionHeader: r['IPv6 Extension Header'] === 'Y', rfc: refs(r.Reference) }))
  write('src/tools/protocol-reference/protocols.json', protocols)

  const icmpCodes = {}
  for (const t of [3, 5, 9, 11, 12, 40, 42, 43]) {
    const f = `icmp-codes-${t}.csv`
    if (existsSync(join(src, f)))
      icmpCodes[t] = csv(f).filter(r => /^\d+$/.test(r.Codes)).map(r => ({ code: Number(r.Codes), name: clean(r.Description) })).filter(c => !/^Unassigned$/i.test(c.name))
  }
  const icmp = csv('icmp-parameters-types.csv')
    .filter(r => /^\d+$/.test(r.Type) && !/^Unassigned$/i.test(r.Name))
    .map(r => ({ type: Number(r.Type), name: clean(r.Name), rfc: refs(r.Reference), codes: icmpCodes[r.Type] ?? [] }))
  // ICMPv6 codes are not published as CSV; curated from RFC 4443, 6554, 7112, 8754, 8883.
  const v6Codes = {
    1: ['No route to destination', 'Communication with destination administratively prohibited', 'Beyond scope of source address', 'Address unreachable', 'Port unreachable', 'Source address failed ingress/egress policy', 'Reject route to destination', 'Error in Source Routing Header', 'Headers too long'],
    3: ['Hop limit exceeded in transit', 'Fragment reassembly time exceeded'],
    4: ['Erroneous header field encountered', 'Unrecognized Next Header type encountered', 'Unrecognized IPv6 option encountered', 'IPv6 First Fragment has incomplete IPv6 Header Chain', 'SR Upper-layer Header Error', 'Unrecognized Next Header type encountered by intermediate node', 'Extension header too big', 'Extension header chain too long', 'Too many extension headers', 'Too many options in extension header', 'Option too big'],
  }
  const icmpv6 = csv('icmpv6-parameters-2.csv')
    .filter(r => /^\d+$/.test(r.Type) && !/^(?:Unassigned|Reserved.*)$/i.test(r.Name))
    .map(r => ({ type: Number(r.Type), name: clean(r.Name), rfc: refs(r.Reference), codes: (v6Codes[r.Type] ?? []).map((name, code) => ({ code, name })) }))
  write('src/tools/protocol-reference/icmp.json', { v4: icmp, v6: icmpv6 })
}

// ── DNS RR types ───────────────────────────────────────────────────────
if (out('src/tools/dns-record-types')) {
  const types = csv('dns-parameters-4.csv')
    .filter(r => /^\d+$/.test(r.Value) && !/^(?:Unassigned|Reserved|Private use)$/i.test(r.TYPE))
    .map(r => ({ type: r.TYPE, value: Number(r.Value), meaning: clean(r.Meaning), rfc: refs(r.Reference) }))
  write('src/tools/dns-record-types/rrtypes.json', types)
}

// ── OUI ────────────────────────────────────────────────────────────────
if (out('src/tools/mac-address-lookup')) {
  const require = createRequire(import.meta.url)
  const oui = require('oui-data')
  const lines = Object.entries(oui)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([prefix, text]) => {
      const parts = text.split('\n').map(clean).filter(Boolean)
      const vendor = parts[0] ?? ''
      const country = parts.length > 1 ? parts.at(-1) : ''
      return `${prefix}\t${vendor}\t${country}`
    })
  write('src/tools/mac-address-lookup/oui.tsv', `${lines.join('\n')}\n`)
}
