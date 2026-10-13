import { expect, test } from '@playwright/test'

test('ipv4-in-ipv6: embeds and extracts', async ({ page }) => {
  await page.goto('/ipv4-in-ipv6?ip=192.0.2.1')
  await expect(page.getByTestId('result')).toContainText('::ffff:192.0.2.1')
  await page.getByTestId('input').fill('2002:c000:201::1')
  await expect(page.getByTestId('result')).toContainText('192.0.2.1')
})
