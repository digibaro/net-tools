import { describe, expect, it } from 'vitest'
import { PROTOCOLS } from './starttls-check.service'

describe('pROTOCOLS', () => {
  it('lists STARTTLS protocols with ports', () => {
    expect(PROTOCOLS).toEqual([{ id: 'smtp', port: 25 }, { id: 'imap', port: 143 }, { id: 'pop3', port: 110 }, { id: 'ftp', port: 21 }, { id: 'xmpp', port: 5222 }])
  })
})
