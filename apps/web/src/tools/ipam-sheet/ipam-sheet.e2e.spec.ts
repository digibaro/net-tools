import { expect, test } from '@playwright/test'

test('ipam-sheet: plan from the URL', async ({ page }) => {
  await page.goto(`/ipam-sheet?block=192.168.10.0/24&plan=${encodeURIComponent('LAN, 10, /25\nVoIP, 20, 50 hosts')}`)
  await expect(page.getByTestId('result')).toContainText('192.168.10.0/25')
  await expect(page.getByTestId('result')).toContainText('192.168.10.128/26')
  await expect(page.getByTestId('result')).toContainText('192.168.10.192/26')
})
