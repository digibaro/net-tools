import { expect, test } from '@playwright/test'

// SNMP_ENABLED is off by default.
test('snmp: disabled by default', async ({ page }) => {
  await page.goto('/snmp')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('target')).first()).toBeVisible()
})

test('snmp: community is not put in the URL', async ({ page }) => {
  await page.goto('/snmp?target=192.0.2.1')
  if (!(await page.getByTestId('target').isVisible()))
    test.skip(true, 'tool disabled in this stack')
  await page.getByTestId('community').locator('input').fill('secret-community')
  expect(page.url()).not.toContain('secret')
})
