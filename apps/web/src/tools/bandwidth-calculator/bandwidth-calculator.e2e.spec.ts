import { expect, test } from '@playwright/test'

test('bandwidth-calculator: 1 GB over 100 Mbit/s without overhead', async ({ page }) => {
  await page.goto('/bandwidth-calculator?size=1&su=GB&rate=100&ru=Mbps&overhead=none&time=80&tu=s')
  await expect(page.getByTestId('time-result').getByTestId('kv-Transfer time')).toHaveText('1 min 20 s')
  await expect(page.getByTestId('rate-result').getByTestId('kv-Required link rate')).toHaveText('100 Mbit/s')
})
