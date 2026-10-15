import { join } from 'node:path'
import { expect, test } from '@playwright/test'

test('pcap-viewer: reads a capture in the browser', async ({ page }) => {
  await page.goto('/pcap-viewer')
  await page.getByTestId('file').setInputFiles(join(import.meta.dirname, 'testdata', 'sample.pcap'))
  await expect(page.getByTestId('result')).toContainText('3')
  await expect(page.getByText('example.com. A').first()).toBeVisible()
  await expect(page.getByTestId('layer')).toHaveCount(4)
})
