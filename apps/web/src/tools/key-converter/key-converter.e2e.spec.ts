import { expect, test } from '@playwright/test'
import { SSH_FIXTURES } from './key-converter.fixtures'

test('key-converter: SSH public key to JWK and PEM', async ({ page }) => {
  await page.goto('/key-converter')
  await page.getByTestId('input').fill(SSH_FIXTURES.ecdsa.pub)
  await expect(page.getByTestId('result')).toContainText(SSH_FIXTURES.ecdsa.fingerprint)
  await expect(page.getByTestId('out-pemSpki')).toContainText(SSH_FIXTURES.ecdsa.spki.split('\n')[1]!)
  await expect(page.getByTestId('out-jwkPublic')).toContainText('"crv": "P-384"')
  expect(page.url()).not.toContain('AAAA')
})
